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

func aiPost(url string, headers map[string]string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
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
		return openaiCall(ctx, c, tenant.AIAPIKey, "https://api.openai.com/v1", model)
	case "custom":
		base := strings.TrimSuffix(tenant.AICustomBaseURL, "/")
		if !isAllowedBaseURL(base) {
			return errors.New("ai: custom base URL must be https (http is allowed only for private-network hosts)")
		}
		if tenant.AICustomModel == "" {
			return errors.New("ai: custom model is not set")
		}
		return openaiCall(ctx, c, tenant.AIAPIKey, base, tenant.AICustomModel)
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

	status, respBody, err := aiPost("https://api.anthropic.com/v1/messages", map[string]string{
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

func openaiCall(ctx context.Context, c *cmd.AIChatCompletion, key, baseURL, model string) error {
	messages := make([]map[string]string, 0, len(c.Messages)+1)
	if c.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": c.System})
	}
	for _, m := range c.Messages {
		messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
	}
	body, err := json.Marshal(map[string]interface{}{
		"model":                 model,
		"max_completion_tokens": c.MaxTokens,
		"messages":              messages,
	})
	if err != nil {
		return errors.Wrap(err, "ai: failed to marshal request")
	}

	status, respBody, err := aiPost(baseURL+"/chat/completions", map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + key,
	}, body)
	if err != nil {
		return errors.Wrap(err, "ai: openai-compatible request failed")
	}
	if status != http.StatusOK {
		return errors.New("ai: provider returned %d: %s", status, truncate(string(respBody), 300))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
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
