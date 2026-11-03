// Package ai relays Vora conversations to the tenant's configured LLM
// provider. Egress is limited to the two known provider hosts plus the
// admin-configured custom base URL (https only); the API key never leaves
// the server. The agent has no tools — the only capability is text.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/errors"
)

func init() {
	bus.Register(Service{})
}

type Service struct{}

func (s Service) Name() string {
	return "AI"
}

func (s Service) Category() string {
	return "ai"
}

func (s Service) Enabled() bool {
	return true
}

func (s Service) Init() {
	bus.AddHandler(chatCompletion)
	bus.AddHandler(webSearch)
}

// LLM calls routinely outlast the 30s global HTTP client (a long finalize on
// a big conversation can take a minute) — providers get their own client.
var aiHTTP = &http.Client{Timeout: 90 * time.Second}

func aiPost(ctx context.Context, url string, headers map[string]string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := aiHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	out, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	return res.StatusCode, out, err
}

// readLimited reads a response body with the same 4MB ceiling as aiPost.
func readLimited(res *http.Response) ([]byte, error) {
	return io.ReadAll(io.LimitReader(res.Body, 4<<20))
}

// UI model selectors map to real model IDs server-side so clients never
// dictate arbitrary models.
var claudeModels = map[string]string{
	"haiku":    "claude-haiku-4-5-20251001",
	"sonnet-5": "claude-sonnet-5",
}

var openaiModels = map[string]string{
	"luna":  "gpt-5.6-luna",
	"terra": "gpt-5.6-terra",
}

// chatCompletion answers c with the tenant's configured provider (Claude,
// OpenAI, or a custom OpenAI-compatible server) and leaves the reply in
// c.Result. MaxTokens outside 1-4000 falls back to 1200.
func chatCompletion(ctx context.Context, c *cmd.AIChatCompletion) error {
	tenant, ok := ctx.Value(app.TenantCtxKey).(*entity.Tenant)
	if !ok || tenant == nil {
		return errors.New("ai: no tenant in context")
	}
	if !tenant.AIEnabled || tenant.AIAPIKey == "" {
		return errors.New("ai: not configured")
	}
	if c.MaxTokens <= 0 || c.MaxTokens > 4000 {
		c.MaxTokens = 1200
	}

	switch tenant.AIProvider {
	case "claude":
		model, ok := claudeModels[tenant.AIModel]
		if !ok {
			model = claudeModels["sonnet-5"]
		}
		return anthropicCall(ctx, c, tenant.AIAPIKey, model)
	case "openai":
		model, ok := openaiModels[tenant.AIModel]
		if !ok {
			model = openaiModels["luna"]
		}
		return openaiCall(ctx, c, tenant.AIAPIKey, "https://api.openai.com/v1", model, nil)
	case "custom":
		base := strings.TrimSuffix(tenant.AICustomBaseURL, "/")
		if !isAllowedBaseURL(base) {
			return errors.New("ai: custom base URL must be https (http is allowed only for private-network hosts)")
		}
		if tenant.AICustomModel == "" {
			return errors.New("ai: custom model is not set")
		}
		// Local servers: Ollama ignores max_completion_tokens (it honours
		// max_tokens), and a thinking model (gemma, qwen) spends most of the
		// budget on hidden reasoning that is thrown away: a finalize ran 5,000+
		// tokens for a 400-token brief and hit the 90s timeout.
		return openaiCall(ctx, c, tenant.AIAPIKey, base, tenant.AICustomModel, map[string]interface{}{
			"max_tokens":       c.MaxTokens,
			"reasoning_effort": "none",
		})
	}
	return errors.New("ai: unknown provider '%s'", tenant.AIProvider)
}

func anthropicCall(ctx context.Context, c *cmd.AIChatCompletion, key, model string) error {
	body, err := json.Marshal(map[string]interface{}{
		"model":      model,
		"max_tokens": c.MaxTokens,
		"system":     c.System,
		"messages":   c.Messages,
	})
	if err != nil {
		return errors.Wrap(err, "ai: failed to marshal request")
	}

	status, respBody, err := aiPost(context.Background(), "https://api.anthropic.com/v1/messages", map[string]string{
		"Content-Type":      "application/json",
		"x-api-key":         key,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return errors.Wrap(err, "ai: anthropic request failed")
	}
	if status != http.StatusOK {
		return errors.New("ai: anthropic returned %d: %s", status, truncate(string(respBody), 300))
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return errors.Wrap(err, "ai: failed to parse anthropic response")
	}
	var sb strings.Builder
	for _, part := range parsed.Content {
		if part.Type == "text" {
			sb.WriteString(part.Text)
		}
	}
	c.Result = strings.TrimSpace(sb.String())
	if c.Result == "" {
		return errors.New("ai: anthropic returned an empty reply")
	}
	return nil
}

// openaiCall sends an OpenAI-style chat completion. Fields in extra are added
// to the request; a server that refuses them (400/422) is asked once more with
// the plain request.
func openaiCall(ctx context.Context, c *cmd.AIChatCompletion, key, baseURL, model string, extra map[string]interface{}) error {
	messages := make([]map[string]string, 0, len(c.Messages)+1)
	if c.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": c.System})
	}
	for _, m := range c.Messages {
		messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
	}
	plain := map[string]interface{}{
		"model":                 model,
		"max_completion_tokens": c.MaxTokens,
		"messages":              messages,
	}
	// One deadline for both attempts, so the fallback never doubles the wait,
	// and a caller that goes away (browser closed) stops the generation.
	deadline, cancel := context.WithTimeout(ctx, aiHTTP.Timeout)
	defer cancel()
	send := func(fields map[string]interface{}) (int, []byte, error) {
		body, err := json.Marshal(fields)
		if err != nil {
			return 0, nil, errors.Wrap(err, "ai: failed to marshal request")
		}
		return aiPost(deadline, baseURL+"/chat/completions", map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + key,
		}, body)
	}

	request := plain
	if len(extra) > 0 {
		request = make(map[string]interface{}, len(plain)+len(extra))
		for k, v := range plain {
			request[k] = v
		}
		for k, v := range extra {
			request[k] = v
		}
	}
	status, respBody, err := send(request)
	if err == nil && len(extra) > 0 && (status == http.StatusBadRequest || status == http.StatusUnprocessableEntity) {
		// Plain on purpose, max_tokens dropped too: if thinking cannot be turned
		// off, a max_tokens cap can be spent entirely on hidden reasoning and
		// leave an empty reply. The previous request is the safe fallback.
		status, respBody, err = send(plain)
	}
	if err != nil {
		return errors.Wrap(err, "ai: openai-compatible request failed")
	}
	if status != http.StatusOK {
		return errors.New("ai: provider returned %d: %s", status, truncate(string(respBody), 300))
	}

	var parsed struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return errors.Wrap(err, "ai: failed to parse provider response")
	}
	if len(parsed.Choices) == 0 {
		return errors.New("ai: provider returned no choices")
	}
	// A reply cut off at the cap would pass as a complete brief or question.
	if parsed.Choices[0].FinishReason == "length" {
		return errors.New("ai: provider stopped at the %d-token limit", c.MaxTokens)
	}
	c.Result = strings.TrimSpace(parsed.Choices[0].Message.Content)
	if c.Result == "" {
		return errors.New("ai: provider returned an empty reply")
	}
	return nil
}

// isAllowedBaseURL mirrors the admin-side validation: https anywhere, plain
// http only toward private/loopback hosts (local LLMs).
func isAllowedBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
