package env

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
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
		// Values that would parse but never match a request's Host, leaving
		// the board unguarded: a trailing dot, an IP literal, https on :80.
		if strings.HasSuffix(u.Hostname(), ".") || net.ParseIP(u.Hostname()) != nil {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' must be a host name without a trailing dot, not an IP address", entry)
		}
		if u.Scheme == "https" && u.Port() == "80" {
			return nil, fmt.Errorf("MCP_ORIGINS: '%s' uses https on port 80", entry)
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
	name, port := host, ""
	if h, p, err := net.SplitHostPort(host); err == nil {
		name, port = h, p
	}
	name = strings.TrimSuffix(name, ".")
	for _, o := range Config.MCPOrigins {
		u, err := url.Parse(o)
		if err != nil || u.Hostname() != name {
			continue
		}
		// The configured port, or none: a request names the default port of
		// whichever scheme the proxy used, or none at all.
		if port == u.Port() || (u.Port() == "" && (port == "443" || port == "80")) {
			return o, true
		}
	}
	return "", false
}

// ValidateMCPPort checks MCP_PORT: the optional dedicated listener for the
// MCP-only addresses. It needs MCP_ORIGINS and a port of its own (not PORT,
// nor the metrics port when metrics are on; pass "" for that otherwise).
func ValidateMCPPort(mcpPort, port, metricsPort string, origins []string) error {
	if mcpPort == "" {
		return nil
	}
	n, err := strconv.Atoi(mcpPort)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("MCP_PORT: '%s' is not a port number", mcpPort)
	}
	if len(origins) == 0 {
		return fmt.Errorf("MCP_PORT needs MCP_ORIGINS: the listener serves only those addresses")
	}
	for name, other := range map[string]string{"PORT": port, "the metrics port": metricsPort} {
		if o, err := strconv.Atoi(other); err == nil && o == n {
			return fmt.Errorf("MCP_PORT must differ from %s (%s)", name, other)
		}
	}
	return nil
}
