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
			From:    c.From.String(),
			To:      []string{to.String()},
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

// send POSTs one rendered message to the Resend API through Fider's shared
// (SSRF-guarded, mockable) HTTP client. On HTTP 429 it retries up to
// maxAttempts, honoring Retry-After; the Idempotency-Key is stable across
// attempts so a retried request Resend already accepted is not delivered twice.
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

		if err := bus.Dispatch(ctx, req); err != nil {
			return err
		}

		if req.ResponseStatusCode >= 200 && req.ResponseStatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("resend returned %d: %s", req.ResponseStatusCode, string(req.ResponseBody))

		// Only 429 is retryable; other 4xx/5xx are returned immediately.
		if req.ResponseStatusCode != http.StatusTooManyRequests || attempt == maxAttempts {
			return lastErr
		}

		wait := retryAfter(req.ResponseHeader)
		log.Warnf(ctx, "Resend rate-limited (attempt @{Attempt}/@{Max}); retrying in @{Wait}.", dto.Props{
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

// retryAfter reads the Retry-After header (seconds), clamped to maxRetryWait.
func retryAfter(h http.Header) time.Duration {
	if h != nil {
		if secs, err := strconv.Atoi(h.Get("Retry-After")); err == nil && secs > 0 {
			wait := time.Duration(secs) * time.Second
			if wait > maxRetryWait {
				return maxRetryWait
			}
			return wait
		}
	}
	return defaultRetryFor
}
