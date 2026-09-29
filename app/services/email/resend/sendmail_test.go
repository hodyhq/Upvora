package resend_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/services/email/resend"
	"github.com/getfider/fider/app/services/httpclient/httpclientmock"

	. "github.com/getfider/fider/app/pkg/assert"
)

var ctx context.Context

func reset() {
	ctx = context.WithValue(context.Background(), app.TenantCtxKey, &entity.Tenant{
		Subdomain: "got",
	})
	httpclientmock.Reset()
	bus.Init(resend.Service{}, httpclientmock.Service{})
}

func TestSend_Success(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	env.Config.Email.Resend.APIKey = "re_test_key"
	reset()

	bus.Publish(ctx, &cmd.SendMail{
		From: dto.Recipient{Name: "Fider Test"},
		To: []dto.Recipient{
			{Name: "Jon Sow", Address: "jon.snow@got.com"},
		},
		TemplateName: "echo_test",
		Props:        dto.Props{"name": "Hello"},
	})

	Expect(httpclientmock.RequestsHistory).HasLen(1)
	Expect(httpclientmock.RequestsHistory[0].URL.String()).Equals("https://api.resend.com/emails")
	Expect(httpclientmock.RequestsHistory[0].Method).Equals("POST")
	Expect(httpclientmock.RequestsHistory[0].Header.Get("Authorization")).Equals("Bearer re_test_key")
	Expect(httpclientmock.RequestsHistory[0].Header.Get("Content-Type")).Equals("application/json")

	body, err := io.ReadAll(httpclientmock.RequestsHistory[0].Body)
	Expect(err).IsNil()

	var payload map[string]interface{}
	Expect(json.Unmarshal(body, &payload)).IsNil()
	Expect(payload["from"]).Equals(`"Fider Test" <noreply@random.org>`)
	Expect(payload["reply_to"]).Equals("noreply@random.org")
	Expect(payload["subject"]).Equals("Message to: Hello")

	to, ok := payload["to"].([]interface{})
	Expect(ok).IsTrue()
	Expect(to).HasLen(1)
	Expect(to[0]).Equals(`"Jon Sow" <jon.snow@got.com>`)
}

// A recipient that can't be sent to (empty or blocklisted) is skipped without
// blocking the rest — the reason the loop uses continue, not return.
func TestSend_SkipsUnsendableRecipient(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	env.Config.Email.Resend.APIKey = "re_test_key"
	reset()

	bus.Publish(ctx, &cmd.SendMail{
		From: dto.Recipient{Name: "Fider Test"},
		To: []dto.Recipient{
			{Name: "Nobody", Address: ""},
			{Name: "Jon Sow", Address: "jon.snow@got.com"},
		},
		TemplateName: "echo_test",
		Props:        dto.Props{"name": "Hello"},
	})

	// Only the valid recipient produced a request.
	Expect(httpclientmock.RequestsHistory).HasLen(1)
	body, err := io.ReadAll(httpclientmock.RequestsHistory[0].Body)
	Expect(err).IsNil()
	var payload map[string]interface{}
	Expect(json.Unmarshal(body, &payload)).IsNil()
	to := payload["to"].([]interface{})
	Expect(to[0]).Equals(`"Jon Sow" <jon.snow@got.com>`)
}

// A 429 is retried (honoring Retry-After) with a stable Idempotency-Key, so a
// rate-limited notification is not silently dropped nor double-sent.
func TestSend_RetriesOn429(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	env.Config.Email.Resend.APIKey = "re_test_key"
	ctx = context.WithValue(context.Background(), app.TenantCtxKey, &entity.Tenant{Subdomain: "got"})

	calls := 0
	keys := []string{}
	bus.Init(resend.Service{})
	bus.AddHandler(func(_ context.Context, c *cmd.HTTPRequest) error {
		calls++
		keys = append(keys, c.Headers["Idempotency-Key"])
		if calls == 1 {
			c.ResponseStatusCode = http.StatusTooManyRequests
			c.ResponseHeader = http.Header{"Retry-After": []string{"1"}}
			c.ResponseBody = []byte("rate limited")
			return nil
		}
		c.ResponseStatusCode = http.StatusOK
		return nil
	})

	bus.Publish(ctx, &cmd.SendMail{
		From:         dto.Recipient{Name: "Fider Test"},
		To:           []dto.Recipient{{Name: "Jon Sow", Address: "jon.snow@got.com"}},
		TemplateName: "echo_test",
		Props:        dto.Props{"name": "Hello"},
	})

	Expect(calls).Equals(2)
	Expect(keys[0]).Equals(keys[1])
	Expect(keys[0]).ContainsSubstring("upvora-")
}

// A transient 5xx is retried the same way as a 429, so a momentary Resend
// outage does not silently drop a notification.
func TestSend_RetriesOn5xx(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	env.Config.Email.Resend.APIKey = "re_test_key"
	ctx = context.WithValue(context.Background(), app.TenantCtxKey, &entity.Tenant{Subdomain: "got"})

	calls := 0
	keys := []string{}
	bus.Init(resend.Service{})
	bus.AddHandler(func(_ context.Context, c *cmd.HTTPRequest) error {
		calls++
		keys = append(keys, c.Headers["Idempotency-Key"])
		if calls == 1 {
			c.ResponseStatusCode = http.StatusInternalServerError
			c.ResponseBody = []byte("boom")
			return nil
		}
		c.ResponseStatusCode = http.StatusOK
		return nil
	})

	bus.Publish(ctx, &cmd.SendMail{
		From:         dto.Recipient{Name: "Fider Test"},
		To:           []dto.Recipient{{Name: "Jon Sow", Address: "jon.snow@got.com"}},
		TemplateName: "echo_test",
		Props:        dto.Props{"name": "Hello"},
	})

	Expect(calls).Equals(2)
	Expect(keys[0]).Equals(keys[1])
}

// A 4xx (bad request / auth) is never retried — retrying an invalid request
// just burns attempts and delays the failure.
func TestSend_DoesNotRetryOn4xx(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	env.Config.Email.Resend.APIKey = "re_test_key"
	ctx = context.WithValue(context.Background(), app.TenantCtxKey, &entity.Tenant{Subdomain: "got"})

	calls := 0
	bus.Init(resend.Service{})
	bus.AddHandler(func(_ context.Context, c *cmd.HTTPRequest) error {
		calls++
		c.ResponseStatusCode = http.StatusUnprocessableEntity
		c.ResponseBody = []byte("invalid from address")
		return nil
	})

	bus.Publish(ctx, &cmd.SendMail{
		From:         dto.Recipient{Name: "Fider Test"},
		To:           []dto.Recipient{{Name: "Jon Sow", Address: "jon.snow@got.com"}},
		TemplateName: "echo_test",
		Props:        dto.Props{"name": "Hello"},
	})

	Expect(calls).Equals(1)
}

// When Retry-After exceeds the synchronous wait budget, we stop rather than
// retry before the window resets (which would just fail again). One attempt,
// no busy-retry.
func TestSend_BailsWhenRetryAfterExceedsBudget(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "multi"
	env.Config.Email.Resend.APIKey = "re_test_key"
	ctx = context.WithValue(context.Background(), app.TenantCtxKey, &entity.Tenant{Subdomain: "got"})

	calls := 0
	bus.Init(resend.Service{})
	bus.AddHandler(func(_ context.Context, c *cmd.HTTPRequest) error {
		calls++
		c.ResponseStatusCode = http.StatusTooManyRequests
		c.ResponseHeader = http.Header{"Retry-After": []string{"30"}}
		c.ResponseBody = []byte("rate limited")
		return nil
	})

	bus.Publish(ctx, &cmd.SendMail{
		From:         dto.Recipient{Name: "Fider Test"},
		To:           []dto.Recipient{{Name: "Jon Sow", Address: "jon.snow@got.com"}},
		TemplateName: "echo_test",
		Props:        dto.Props{"name": "Hello"},
	})

	Expect(calls).Equals(1)
}
