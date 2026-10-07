package web

import (
	"context"
	"net/http"
	"path"
	"strings"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/oauthas"
)

// Paths an MCP-only public address serves: the MCP endpoint, OAuth and its
// discovery, and what the consent page needs to render (the app's code bundles,
// built-in images and the site logo). Everything else is a plain 404, so the
// board itself stays internal. An allowlist fails closed: a route nobody listed
// is never public by accident.
var (
	mcpOriginPaths = map[string]bool{
		"/mcp": true,
		"/.well-known/oauth-authorization-server":   true,
		"/.well-known/oauth-protected-resource":     true,
		"/.well-known/oauth-protected-resource/mcp": true,
		"/oauth2/register":                          true,
		"/oauth2/token":                             true,
		"/oauth2/authorize":                         true,
		"/oauth2/resume":                            true,
		"/_api/oauth2/authorize":                    true,
		"/static/favicon":                           true,
	}
	mcpOriginPrefixes = []string{
		"/assets/",
		"/static/assets/",
		"/static/images/logos/",
		"/static/favicon/logos/",
	}
)

// requestOrigin is scheme://host as the client addressed it, after the
// reverse proxy's X-Forwarded-Proto and X-Forwarded-Host. On an MCP-only
// address it is always that address as configured, whatever the scheme the
// proxy reported, so the gate fails closed and emitted URLs stay https.
func requestOrigin(r *http.Request) string {
	if origin, ok := mcpOriginOf(r); ok {
		return origin
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + host
}

func mcpOriginAllows(p string) bool {
	if p != path.Clean(p) {
		return false
	}
	if mcpOriginPaths[p] {
		return true
	}
	for _, prefix := range mcpOriginPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// MCPOriginGate answers 404 for every path outside the MCP surface when a
// request arrives on one of the MCP_ORIGINS (or on the MCP listener, where
// nothing is served if none is configured). In-process tool replays (which
// keep the caller's host) may call the REST API; other addresses are untouched.
func MCPOriginGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, onMCP := mcpOriginOf(r)
		if onMCP || onMCPListener(r) {
			// In-process tool replays only ever call the REST API.
			replay, _ := r.Context().Value(oauthas.ReplayCtxKey{}).(bool)
			allowed := mcpOriginAllows(r.URL.Path) || (replay && strings.HasPrefix(r.URL.Path, "/api/"))
			if !onMCP || !allowed {
				http.NotFound(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// mcpOriginOf reports whether any host the request names, the connection's
// Host or any X-Forwarded-Host value, is an MCP-only address. Any one is
// enough: a client on the public MCP-only address must not reach the board by
// claiming the board's host in X-Forwarded-Host, which some proxies pass on.
func mcpOriginOf(r *http.Request) (string, bool) {
	if len(env.Config.MCPOrigins) == 0 {
		return "", false
	}
	hosts := []string{r.Host}
	for _, fwd := range r.Header.Values("X-Forwarded-Host") {
		hosts = append(hosts, strings.Split(fwd, ",")...)
	}
	for _, h := range hosts {
		if origin, ok := env.MCPOriginForHost(h); ok {
			return origin, true
		}
	}
	// On the dedicated listener the headers prove nothing either way.
	if onMCPListener(r) {
		return env.Config.MCPOrigins[0], true
	}
	return "", false
}

type mcpListenerCtxKey struct{}

func onMCPListener(r *http.Request) bool {
	on, _ := r.Context().Value(mcpListenerCtxKey{}).(bool)
	return on
}

// MCPListenerHandler serves the dedicated MCP listener (MCP_PORT). Every
// request on it is MCP-only whatever its Host or forwarded headers say, so a
// proxy that rewrites Host, or a client that sends any Host, cannot reach the
// board through it. Point the public tunnel at this port.
func MCPListenerHandler(next http.Handler) http.Handler {
	gated := MCPOriginGate(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gated.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), mcpListenerCtxKey{}, true)))
	})
}
