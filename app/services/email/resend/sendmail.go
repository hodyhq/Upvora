package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/log"
	"github.com/getfider/fider/app/pkg/rand"
	"github.com/getfider/fider/app/services/email"
)

// Resend rate-limits (2 req/s by default); a burst fan-out can hit 429. Retry
// a bounded number of times, honoring Retry-After, so a single 429 does not
// silently drop a notification.
const (
	maxAttempts     = 3
	maxRetryWait    = 5 * time.Second
	defaultRetryFor = time.Second
)

// resendPayload is the Resend "send email" request body.
// https://resend.com/docs/api-reference/emails/send-email
type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	ReplyTo string   `json:"reply_to,omitempty"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Tags    []tag    `json:"tags,omitempty"`
}

type tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// sendMail delivers each recipient's message through the Resend API. One
// request per recipient (mirrors the SMTP provider); a single failed or
// skipped recipient never blocks the rest.
func sendMail(ctx context.Context, c *cmd.SendMail) {
	if c.Props == nil {
		c.Props = dto.Props{}
	}

	if c.From.Address == "" {
		c.From.Address = email.NoReply
	}

	for _, to := range c.To {
		if to.Address == "" {
			continue
		}

		if !email.CanSendTo(to.Address) {
			log.Warnf(ctx, "Skipping email to '@{Name} <@{Address}>'.", dto.Props{
				"Name":    to.Name,
				"Address": to.Address,
			})
			continue
		}

		log.Debugf(ctx, "Sending email to @{Address} with template @{TemplateName} and params @{Props}.", dto.Props{
			"Address":      to.Address,
			"TemplateName": c.TemplateName,
			"Props":        to.Props,
		})

		message := email.RenderMessage(ctx, c.TemplateName, c.From.Address, c.Props.Merge(to.Props))

		payload := resendPayload{
			From:    resendAddress(c.From),
			To:      []string{resendAddress(to)},
			ReplyTo: c.From.Address,
			Subject: email.EncodeSubject(message.Subject),
			HTML:    message.Body,
			Tags:    []tag{{Name: "template", Value: c.TemplateName}},
		}

		if err := send(ctx, payload); err != nil {
			log.Errorf(ctx, "Failed to send email to @{Address}: @{Error}", dto.Props{
				"Address": to.Address,
				"Error":   err.Error(),
			})
			continue
		}
	}
}

// resendAddress renders a recipient the way Resend's API accepts it: a bare
// "email@example.com" when there is no display name, or "Name <email@…>" when
// there is. Go's net/mail renders an empty-name address as "<email@…>", which
// Resend rejects with a 422 validation_error — and most transactional mail
// (sign-in codes, notifications) goes to a bare address with no name.
func resendAddress(r dto.Recipient) string {
	if r.Name == "" {
		return r.Address
	}
	return r.String()
}

// send POSTs one rendered message to the Resend API through Fider's shared
// (SSRF-guarded, mockable) HTTP client. Transient failures — a transport
// error, HTTP 429, or any 5xx — are retried up to maxAttempts; the
// Idempotency-Key is stable across attempts so a request Resend already
// accepted is not delivered twice. When a 429 asks for longer than we are
// willing to wait synchronously, we stop rather than retry before the window
// resets (which would just fail again).
func send(ctx context.Context, payload resendPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// One key per logical message, reused across retries.
	idempotencyKey := "upvora-" + rand.String(32)

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req := &cmd.HTTPRequest{
			Method: "POST",
			URL:    endpoint,
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Content-Type":    "application/json",
				"Authorization":   "Bearer " + env.Config.Email.Resend.APIKey,
				"Idempotency-Key": idempotencyKey,
			},
		}

		dispatchErr := bus.Dispatch(ctx, req)

		var wait time.Duration
		retryable := false
		switch {
		case dispatchErr != nil:
			// Transport-level failure (timeout, connection reset) — transient.
			lastErr = dispatchErr
			retryable = true
			wait = defaultRetryFor
		case req.ResponseStatusCode >= 200 && req.ResponseStatusCode < 300:
			return nil
		case req.ResponseStatusCode == http.StatusTooManyRequests:
			lastErr = fmt.Errorf("resend returned %d: %s", req.ResponseStatusCode, string(req.ResponseBody))
			retryable = true
			wait = retryAfter(req.ResponseHeader)
		case req.ResponseStatusCode >= 500:
			// Transient server error.
			lastErr = fmt.Errorf("resend returned %d: %s", req.ResponseStatusCode, string(req.ResponseBody))
			retryable = true
			wait = defaultRetryFor
		default:
			// 4xx (bad request, auth, validation) — not retryable.
			return fmt.Errorf("resend returned %d: %s", req.ResponseStatusCode, string(req.ResponseBody))
		}

		// Out of attempts, or the required wait exceeds what we will block on:
		// retrying early would just fail again before the limit resets.
		if !retryable || attempt == maxAttempts || wait > maxRetryWait {
			return lastErr
		}

		log.Warnf(ctx, "Resend transient failure (attempt @{Attempt}/@{Max}); retrying in @{Wait}.", dto.Props{
			"Attempt": attempt, "Max": maxAttempts, "Wait": wait.String(),
		})
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

// retryAfter reads the Retry-After header (delay in seconds). It returns the
// delay the server actually asked for — the caller decides whether that is
// within its synchronous wait budget.
func retryAfter(h http.Header) time.Duration {
	if h != nil {
		if secs, err := strconv.Atoi(h.Get("Retry-After")); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultRetryFor
}
