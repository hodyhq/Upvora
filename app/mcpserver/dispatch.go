package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/oauthas"
)

const (
	maxOutputBytes = 64 * 1024
	maxListLimit   = 100
	replayTimeout  = 60 * time.Second
)

// Field is one tool argument: a path segment, a query parameter (GET), or a
// JSON body field (other methods).
type Field struct {
	Name        string
	Type        string // integer, number, string, boolean, array, object
	In          string // path, query, body, both (path and body), raw (the whole body)
	Description string
	Required    bool
	Items       string // element type for arrays
}

// Tool maps an MCP tool onto one /api/v1 route. The route's own middleware
// decides authorization; MinRole and the flags only hide tools a user could
// never call.
type Tool struct {
	Name        string
	Description string
	Method      string
	Path        string // /api/v1 template with {param} segments
	MinRole     enum.Role
	Fields      []Field
	Paged       bool // has a "limit" query parameter capped at maxListLimit
	Pro         bool // requires a pro (or self-hosted) site
	Billing     bool // only when billing is enabled
	MultiTenant bool // only on hosted multi-tenant installs
}

// Visible reports whether user may see tool on tenant with a token of scope.
func Visible(t Tool, user *entity.User, tenant *entity.Tenant, scope string) bool {
	if user == nil || user.Role < t.MinRole {
		return false
	}
	if scope == oauthas.ScopeRead && t.Method != http.MethodGet {
		return false
	}
	if t.Billing && !env.IsBillingEnabled() {
		return false
	}
	if t.MultiTenant && env.IsSingleHostMode() {
		return false
	}
	if t.Pro && (tenant == nil || !tenant.IsPro) {
		return false
	}
	return true
}

// InputSchema is the tool's JSON Schema.
func (t Tool) InputSchema() map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, f := range t.Fields {
		p := map[string]any{"type": f.Type}
		if f.Description != "" {
			p["description"] = f.Description
		}
		if f.Type == "array" {
			item := f.Items
			if item == "" {
				item = "string"
			}
			p["items"] = map[string]any{"type": item}
		}
		props[f.Name] = p
		if f.Required {
			required = append(required, f.Name)
		}
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// Result is the outcome of a replayed call.
type Result struct {
	Text    string
	IsError bool
}

func errResult(format string, a ...any) Result {
	return Result{Text: fmt.Sprintf(format, a...), IsError: true}
}

func argString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case int:
		return strconv.Itoa(x), true
	case bool:
		return strconv.FormatBool(x), true
	default:
		return "", false
	}
}

// Dispatch replays a tool call in-process against /api/v1 through engine, as
// the caller (same Authorization and host headers as the /mcp request).
func Dispatch(engine http.Handler, orig *http.Request, t Tool, args map[string]any) Result {
	return DispatchContext(context.Background(), engine, orig, t, args)
}

// DispatchContext is Dispatch bound to the tool call's context, so a client
// that goes away cancels the replayed request.
func DispatchContext(parent context.Context, engine http.Handler, orig *http.Request, t Tool, args map[string]any) Result {
	path := t.Path
	query := url.Values{}
	body := map[string]any{}
	var rawBody any // a field sent as the whole body (e.g. a JSON array)

	for _, f := range t.Fields {
		v, present := args[f.Name]
		if !present || v == nil {
			if f.Required {
				return errResult("Missing required argument %q.", f.Name)
			}
			continue
		}
		switch f.In {
		case "path", "both":
			s, ok := argString(v)
			// Path templates are fixed; a value may never add or climb segments
			// (the router decodes %2F back to "/").
			if !ok || s == "" || s == "." || strings.Contains(s, "..") || strings.ContainsAny(s, "/\\") {
				return errResult("Invalid value for %q.", f.Name)
			}
			if f.Type == "integer" && !isDigits(s) {
				return errResult("%q must be a whole number.", f.Name)
			}
			path = strings.Replace(path, "{"+f.Name+"}", url.PathEscape(s), 1)
			if f.In == "both" {
				body[f.Name] = v
			}
		case "raw":
			rawBody = v
		case "query":
			s, ok := argString(v)
			if !ok {
				return errResult("Invalid value for %q.", f.Name)
			}
			query.Set(f.Name, s)
		default:
			body[f.Name] = v
		}
	}
	if t.Paged && query.Has("limit") {
		// "all", negatives and anything over the cap become the cap.
		if n, err := strconv.Atoi(query.Get("limit")); err != nil || n < 1 || n > maxListLimit {
			query.Set("limit", strconv.Itoa(maxListLimit))
		}
	}
	if strings.Contains(path, "{") {
		return errResult("Missing path argument for %s.", t.Name)
	}

	var reader *bytes.Reader
	if t.Method == http.MethodGet || t.Method == http.MethodDelete && len(body) == 0 {
		reader = bytes.NewReader(nil)
	} else {
		var payload any = body
		if rawBody != nil {
			payload = rawBody
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return errResult("Arguments could not be encoded.")
		}
		reader = bytes.NewReader(raw)
	}

	ctx, cancel := context.WithTimeout(parent, replayTimeout)
	defer cancel()
	target := "http://" + orig.Host + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, t.Method, target, reader)
	if err != nil {
		return errResult("Request could not be built.")
	}
	req.Host = orig.Host
	// Fider reads the path from RequestURI, which only an HTTP server sets.
	req.RequestURI = req.URL.RequestURI()
	for _, h := range []string{"Authorization", "X-Forwarded-Proto", "X-Forwarded-Host", "X-Forwarded-For", "CF-Connecting-IP"} {
		if v := orig.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	if orig.TLS != nil && req.Header.Get("X-Forwarded-Proto") == "" {
		req.Header.Set("X-Forwarded-Proto", "https")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	text := rec.Body.String()
	if len(text) > maxOutputBytes {
		text = text[:maxOutputBytes] + "\n[output truncated at 64 KB; narrow the request]"
	}
	if rec.Code >= 400 {
		return Result{Text: fmt.Sprintf("HTTP %d: %s", rec.Code, text), IsError: true}
	}
	if text == "" {
		text = "{}"
	}
	return Result{Text: text}
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
