package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	. "github.com/getfider/fider/app/pkg/assert"
)

// fakeProvider is an OpenAI-compatible /chat/completions endpoint that records
// every request body and answers with reply(body). seen() returns the bodies
// received so far; a handler can still be running after the client gave up.
func fakeProvider(t *testing.T, reply func(body map[string]interface{}) (int, string)) (*httptest.Server, func() []map[string]interface{}) {
	var mu sync.Mutex
	bodies := []map[string]interface{}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]interface{}{}
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		status, out := reply(body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]interface{}{}, bodies...)
	}
}

func customTenantCtx(baseURL string) context.Context {
	return context.WithValue(context.Background(), app.TenantCtxKey, &entity.Tenant{
		AIEnabled: true, AIAPIKey: "k", AIProvider: "custom",
		AICustomBaseURL: baseURL, AICustomModel: "gemma4:12b-it-qat",
	})
}

const okReply = `{"choices":[{"message":{"content":"Hello."}}]}`

// Local servers: Ollama ignores max_completion_tokens (it honours max_tokens),
// and thinking models spend most of the budget, and minutes, on hidden
// reasoning that is thrown away. Both fields go on a custom provider.
func TestChatCompletion_CustomProvider_CapsTokensAndTurnsThinkingOff(t *testing.T) {
	RegisterT(t)
	srv, seen := fakeProvider(t, func(map[string]interface{}) (int, string) { return 200, okReply })

	c := &cmd.AIChatCompletion{Messages: []entity.AIMessage{{Role: "user", Content: "hi"}}, MaxTokens: 2500}
	Expect(chatCompletion(customTenantCtx(srv.URL), c)).IsNil()

	Expect(c.Result).Equals("Hello.")
	Expect(len(seen())).Equals(1)
	body := seen()[0]
	Expect(body["max_tokens"]).Equals(float64(2500))
	Expect(body["max_completion_tokens"]).Equals(float64(2500))
	Expect(body["reasoning_effort"]).Equals("none")
}

// A server that refuses those fields gets exactly the request it got before.
func TestChatCompletion_CustomProvider_RetriesPlainWhenRefused(t *testing.T) {
	RegisterT(t)
	srv, seen := fakeProvider(t, func(body map[string]interface{}) (int, string) {
		if _, ok := body["reasoning_effort"]; ok {
			return 400, `{"error":"unknown field reasoning_effort"}`
		}
		return 200, okReply
	})

	c := &cmd.AIChatCompletion{Messages: []entity.AIMessage{{Role: "user", Content: "hi"}}, MaxTokens: 2500}
	Expect(chatCompletion(customTenantCtx(srv.URL), c)).IsNil()

	Expect(c.Result).Equals("Hello.")
	if !Expect(len(seen())).Equals(2) {
		return
	}
	plain := seen()[1]
	_, hasEffort := plain["reasoning_effort"]
	_, hasMaxTokens := plain["max_tokens"]
	Expect(hasEffort).IsFalse()
	Expect(hasMaxTokens).IsFalse()
	Expect(plain["max_completion_tokens"]).Equals(float64(2500))
}

// The fallback shares the first attempt's deadline: one completion never
// waits longer than one client timeout, however slow the refusal was.
func TestChatCompletion_CustomProvider_RetryKeepsTheOneDeadline(t *testing.T) {
	RegisterT(t)
	saved := aiHTTP
	// Each attempt alone fits the timeout; both together (800ms) do not.
	aiHTTP = &http.Client{Timeout: 600 * time.Millisecond}
	defer func() { aiHTTP = saved }()
	srv, seen := fakeProvider(t, func(body map[string]interface{}) (int, string) {
		time.Sleep(400 * time.Millisecond)
		if _, ok := body["reasoning_effort"]; ok {
			return 400, `{"error":"unknown field reasoning_effort"}`
		}
		return 200, okReply
	})

	c := &cmd.AIChatCompletion{Messages: []entity.AIMessage{{Role: "user", Content: "hi"}}, MaxTokens: 100}
	start := time.Now()
	err := chatCompletion(customTenantCtx(srv.URL), c)

	Expect(err).IsNotNil()
	Expect(time.Since(start) < 790*time.Millisecond).IsTrue()
	Expect(len(seen()) >= 1).IsTrue()
}

// Any other failure is reported once, without a second request.
func TestChatCompletion_CustomProvider_DoesNotRetryServerErrors(t *testing.T) {
	RegisterT(t)
	srv, seen := fakeProvider(t, func(map[string]interface{}) (int, string) { return 500, `{"error":"boom"}` })

	c := &cmd.AIChatCompletion{Messages: []entity.AIMessage{{Role: "user", Content: "hi"}}, MaxTokens: 100}
	err := chatCompletion(customTenantCtx(srv.URL), c)

	Expect(err).IsNotNil()
	Expect(len(seen())).Equals(1)
}
