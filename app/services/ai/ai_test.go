package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	. "github.com/getfider/fider/app/pkg/assert"
)

// fakeProvider is an OpenAI-compatible /chat/completions endpoint that records
// every request body and answers with reply(body).
func fakeProvider(t *testing.T, reply func(body map[string]interface{}) (int, string)) (*httptest.Server, *[]map[string]interface{}) {
	seen := []map[string]interface{}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]interface{}{}
		_ = json.Unmarshal(raw, &body)
		seen = append(seen, body)
		status, out := reply(body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
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
	Expect(len(*seen)).Equals(1)
	body := (*seen)[0]
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
	if !Expect(len(*seen)).Equals(2) {
		return
	}
	plain := (*seen)[1]
	_, hasEffort := plain["reasoning_effort"]
	_, hasMaxTokens := plain["max_tokens"]
	Expect(hasEffort).IsFalse()
	Expect(hasMaxTokens).IsFalse()
	Expect(plain["max_completion_tokens"]).Equals(float64(2500))
}

// Any other failure is reported once, without a second request.
func TestChatCompletion_CustomProvider_DoesNotRetryServerErrors(t *testing.T) {
	RegisterT(t)
	srv, seen := fakeProvider(t, func(map[string]interface{}) (int, string) { return 500, `{"error":"boom"}` })

	c := &cmd.AIChatCompletion{Messages: []entity.AIMessage{{Role: "user", Content: "hi"}}, MaxTokens: 100}
	err := chatCompletion(customTenantCtx(srv.URL), c)

	Expect(err).IsNotNil()
	Expect(len(*seen)).Equals(1)
}
