package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/oauthas"
	"github.com/getfider/fider/app/pkg/web"
)

func gateStatus(target string, headers map[string]string, ctx context.Context) int {
	gate := web.MCPOriginGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest("GET", target, nil)
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	return rec.Code
}

func TestMCPOriginGate_ServesOnlyTheMCPSurface(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()

	for _, path := range []string{
		"/mcp",
		"/.well-known/oauth-authorization-server",
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/mcp",
		"/oauth2/register", "/oauth2/token", "/oauth2/authorize", "/oauth2/resume",
		"/_api/oauth2/authorize",
		"/assets/js/main.js", "/static/assets/logo.png", "/static/favicon",
		"/static/images/logos/abc-logo.png", "/static/favicon/logos/abc-logo.png",
	} {
		Expect(gateStatus("https://mcp.example.com"+path, nil, nil)).Equals(http.StatusTeapot)
	}

	for _, path := range []string{
		"/", "/admin", "/admin/members", "/posts/1", "/roadmap", "/signin", "/signup",
		"/api/v1/posts", "/api/v1/users", "/_api/admin/settings", "/oauth2/continue",
		"/static/images/attachments/secret.png", "/static/images/logos/../attachments/secret.png",
		"/static/avatars/letter/1/a", "/robots.txt", "/_health", "/feed/global.atom",
		"/sitemap.xml", "/oauth/google/callback", "/does-not-exist",
	} {
		Expect(gateStatus("https://mcp.example.com"+path, nil, nil)).Equals(http.StatusNotFound)
	}
}

func TestMCPOriginGate_UsesForwardedHost(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()

	fwd := map[string]string{"X-Forwarded-Host": "mcp.example.com", "X-Forwarded-Proto": "https"}
	Expect(gateStatus("http://upvora:3000/admin", fwd, nil)).Equals(http.StatusNotFound)
	Expect(gateStatus("http://upvora:3000/mcp", fwd, nil)).Equals(http.StatusTeapot)
}

func TestMCPOriginGate_OtherOriginsAndReplaysUntouched(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()

	Expect(gateStatus("https://ideas.internal.example/admin", nil, nil)).Equals(http.StatusTeapot)
	// MCP tool calls replay /api/v1 in-process with the caller's host.
	replay := context.WithValue(context.Background(), oauthas.ReplayCtxKey{}, true)
	Expect(gateStatus("https://mcp.example.com/api/v1/posts", nil, replay)).Equals(http.StatusTeapot)
}

func TestMCPOriginGate_NoOriginsConfiguredIsANoOp(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = nil
	Expect(gateStatus("https://anything.example.com/admin", nil, nil)).Equals(http.StatusTeapot)
}

// A proxy that forwards over plain HTTP without X-Forwarded-Proto must not
// open the board on the MCP-only address: the host alone decides (fail closed).
func TestMCPOriginGate_HostDecidesEvenWithoutForwardedProto(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()

	Expect(gateStatus("http://mcp.example.com/admin", nil, nil)).Equals(http.StatusNotFound)
	Expect(gateStatus("http://mcp.example.com:80/admin", nil, nil)).Equals(http.StatusNotFound)
	Expect(gateStatus("http://upvora:3000/admin", map[string]string{"X-Forwarded-Host": "MCP.example.com"}, nil)).Equals(http.StatusNotFound)
	Expect(gateStatus("http://mcp.example.com/mcp", nil, nil)).Equals(http.StatusTeapot)
}

// On the MCP-only address the request is always its configured origin, so
// every URL Upvora emits there (issuer, endpoints, audience) is the https one.
func TestWrapRequest_MCPOriginIsCanonical(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()

	req := httptest.NewRequest("GET", "http://mcp.example.com/.well-known/oauth-authorization-server", nil)
	req.RequestURI = req.URL.RequestURI()
	wrapped := web.WrapRequest(req)
	Expect(wrapped.BaseURL()).Equals("https://mcp.example.com")
	Expect(wrapped.IsSecure).IsTrue()
}

// A client on the public MCP-only address cannot reach the board by claiming
// the board's host in X-Forwarded-Host (proxies such as Cloudflare pass it on).
func TestMCPOriginGate_ForwardedHostCannotEscape(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()

	escape := map[string]string{"X-Forwarded-Host": "ideas.internal.example", "X-Forwarded-Proto": "https"}
	Expect(gateStatus("https://mcp.example.com/admin", escape, nil)).Equals(http.StatusNotFound)
	list := map[string]string{"X-Forwarded-Host": "ideas.internal.example, mcp.example.com"}
	Expect(gateStatus("http://upvora:3000/admin", list, nil)).Equals(http.StatusNotFound)

	req := httptest.NewRequest("GET", "https://mcp.example.com/mcp", nil)
	req.Header.Set("X-Forwarded-Host", "ideas.internal.example")
	req.RequestURI = req.URL.RequestURI()
	wrapped := web.WrapRequest(req)
	Expect(wrapped.BaseURL()).Equals("https://mcp.example.com")
}
