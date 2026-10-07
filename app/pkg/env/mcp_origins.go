package env

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseMCPOrigins validates MCP_ORIGINS: extra public addresses that serve
// only the MCP connection (the MCP endpoint, OAuth and discovery), so a board
// can stay internal while assistants reach it. Each entry must be an absolute
// https origin (http only in development), with no path, query, fragment,
// credentials or wildcard, distinct from BASE_URL and from each other.
func ParseMCPOrigins(raw, baseURL string, allowHTTP bool) ([]string, error) {
	var origins []string
	seen := map[string]bool{}
	base, _ := normalizeOrigin(baseURL)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		u, err := url.Parse(entry)
		if err != nil || u.Host == "" || !u.IsAbs() {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' is not an absolute origin such as https://mcp.example.com", entry)
		}
		if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || strings.Contains(entry, "#") || strings.Contains(entry, "?") || u.User != nil {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' must be an origin only, with no path, query, fragment or credentials", entry)
		}
		if strings.Contains(u.Host, "*") {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' uses a wildcard; list each address exactly", entry)
		}
		if u.Scheme != "https" && !(allowHTTP && u.Scheme == "http") {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' must use https", entry)
		}
		origin, _ := normalizeOrigin(entry)
		if origin == base {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' is the same as BASE_URL; list only the extra public addresses", entry)
		}
		if seen[origin] {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' is listed twice", entry)
		}
		seen[origin] = true
		origins = append(origins, origin)
	}
	return origins, nil
}

// normalizeOrigin lowercases scheme and host and drops the default port, so
// "https://MCP.example.com:443/" and "https://mcp.example.com" compare equal.
func normalizeOrigin(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host, true
}

// IsMCPOrigin reports whether origin (scheme://host[:port]) is one of the
// configured MCP-only public addresses.
func IsMCPOrigin(origin string) bool {
	_, ok := MatchMCPOrigin(origin)
	return ok
}

// MatchMCPOrigin returns the configured MCP-only address that origin names
// (in its canonical form), if any.
func MatchMCPOrigin(origin string) (string, bool) {
	if len(Config.MCPOrigins) == 0 {
		return "", false
	}
	normalized, ok := normalizeOrigin(origin)
	if !ok {
		return "", false
	}
	for _, o := range Config.MCPOrigins {
		if o == normalized {
			return o, true
		}
	}
	return "", false
}

// MCPOriginForHost returns the configured MCP-only address whose host (and
// port, default ports ignored) is host, whatever scheme the request came in
// on. Matching on the host alone fails closed: a proxy that forwards over
// plain HTTP without X-Forwarded-Proto still gets the MCP-only treatment.
func MCPOriginForHost(host string) (string, bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || len(Config.MCPOrigins) == 0 {
		return "", false
	}
	host = strings.TrimSuffix(strings.TrimSuffix(host, ":443"), ":80")
	for _, o := range Config.MCPOrigins {
		if strings.SplitN(o, "://", 2)[1] == host {
			return o, true
		}
	}
	return "", false
}
