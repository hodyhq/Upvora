package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/pkg/errors"
)

// maxWebResults caps how many hits Vora ever sees — enough to ground an
// answer, small enough to stay cheap and keep the prompt tight.
const maxWebResults = 6

// webSearch dispatches to the tenant's configured search provider. Results are
// returned as plain data for Vora to reference; they are never trusted as
// instructions (the handler labels them as untrusted context).
func webSearch(ctx context.Context, c *cmd.AIWebSearch) error {
	tenant, ok := ctx.Value(app.TenantCtxKey).(*entity.Tenant)
	if !ok || tenant == nil {
		return errors.New("ai: no tenant in context")
	}
	if !tenant.AIWebSearchEnabled {
		return errors.New("ai: web search not configured")
	}
	query := strings.TrimSpace(c.Query)
	if query == "" {
		return nil
	}

	switch tenant.AIWebSearchProvider {
	case "serper":
		return serperSearch(c, query, tenant.AIWebSearchAPIKey)
	case "searxng":
		return searxngSearch(c, query, tenant.AIWebSearchBaseURL)
	}
	return errors.New("ai: unknown web search provider '%s'", tenant.AIWebSearchProvider)
}

func serperSearch(c *cmd.AIWebSearch, query, key string) error {
	if key == "" {
		return errors.New("ai: serper api key is not set")
	}
	body, err := json.Marshal(map[string]interface{}{"q": query, "num": maxWebResults})
	if err != nil {
		return errors.Wrap(err, "ai: failed to marshal serper request")
	}
	status, respBody, err := aiPost("https://google.serper.dev/search", map[string]string{
		"Content-Type": "application/json",
		"X-API-KEY":    key,
	}, body)
	if err != nil {
		return errors.Wrap(err, "ai: serper request failed")
	}
	if status != http.StatusOK {
		return errors.New("ai: serper returned %d: %s", status, truncate(string(respBody), 200))
	}

	var parsed struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return errors.Wrap(err, "ai: failed to parse serper response")
	}
	for _, r := range parsed.Organic {
		c.Result = append(c.Result, cmd.AIWebSearchResult{Title: r.Title, URL: r.Link, Snippet: r.Snippet})
		if len(c.Result) >= maxWebResults {
			break
		}
	}
	return nil
}

func searxngSearch(c *cmd.AIWebSearch, query, baseURL string) error {
	base := strings.TrimSuffix(baseURL, "/")
	// Same SSRF guard as the custom LLM base URL: https anywhere, http only
	// toward loopback/private hosts (a self-hosted SearXNG on the LAN).
	if !isAllowedBaseURL(base) {
		return errors.New("ai: searxng URL must be https (http is allowed only for private-network hosts)")
	}
	endpoint := base + "/search?" + url.Values{
		"q":      {query},
		"format": {"json"},
	}.Encode()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return errors.Wrap(err, "ai: failed to build searxng request")
	}
	req.Header.Set("Accept", "application/json")
	res, err := aiHTTP.Do(req)
	if err != nil {
		return errors.Wrap(err, "ai: searxng request failed")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return errors.New("ai: searxng returned %d", res.StatusCode)
	}
	respBody, err := readLimited(res)
	if err != nil {
		return errors.Wrap(err, "ai: failed to read searxng response")
	}

	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return errors.Wrap(err, "ai: failed to parse searxng response")
	}
	for _, r := range parsed.Results {
		c.Result = append(c.Result, cmd.AIWebSearchResult{Title: r.Title, URL: r.URL, Snippet: r.Content})
		if len(c.Result) >= maxWebResults {
			break
		}
	}
	return nil
}
