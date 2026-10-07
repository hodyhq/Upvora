package env_test

import (
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/env"
)

func TestParseMCPOrigins_Valid(t *testing.T) {
	RegisterT(t)
	origins, err := env.ParseMCPOrigins(" https://MCP.Example.com , https://other.example.com:8443 ", "https://ideas.internal.example", false)
	Expect(err).IsNil()
	Expect(origins).Equals([]string{"https://mcp.example.com", "https://other.example.com:8443"})

	origins, err = env.ParseMCPOrigins("", "https://ideas.internal.example", false)
	Expect(err).IsNil()
	Expect(len(origins)).Equals(0)

	// The default port is dropped so it matches how requests report it.
	origins, _ = env.ParseMCPOrigins("https://mcp.example.com:443", "https://ideas.internal.example", false)
	Expect(origins).Equals([]string{"https://mcp.example.com"})
}

func TestParseMCPOrigins_Invalid(t *testing.T) {
	RegisterT(t)
	for _, raw := range []string{
		"mcp.example.com",                             // not absolute
		"https://mcp.example.com/mcp",                 // has a path
		"https://mcp.example.com?x=1",                 // has a query
		"https://mcp.example.com#f",                   // has a fragment
		"https://user@mcp.example.com",                // has credentials
		"http://mcp.example.com",                      // not https
		"https://*.example.com",                       // wildcard
		"https://ideas.internal.example",              // same as BASE_URL
		"https://IDEAS.internal.example:443/",         // same as BASE_URL after normalising
		"https://a.example.com,https://A.example.com", // duplicate
		"https://mcp.example.com.",                    // trailing dot never matches a Host
		"https://[2001:db8::1]",                       // IP literal
		"https://203.0.113.7",                         // IP literal
		"https://mcp.example.com:80",                  // https on the http port
	} {
		_, err := env.ParseMCPOrigins(raw, "https://ideas.internal.example", false)
		Expect(err).IsNotNil()
	}
}

func TestParseMCPOrigins_HTTPOnlyInDevelopment(t *testing.T) {
	RegisterT(t)
	origins, err := env.ParseMCPOrigins("http://mcp.localhost:3000", "http://localhost:3000", true)
	Expect(err).IsNil()
	Expect(origins).Equals([]string{"http://mcp.localhost:3000"})
}

func TestIsMCPOrigin(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com"}
	defer func() { env.Config.MCPOrigins = nil }()
	Expect(env.IsMCPOrigin("https://mcp.example.com")).IsTrue()
	Expect(env.IsMCPOrigin("https://MCP.example.com:443")).IsTrue()
	Expect(env.IsMCPOrigin("http://mcp.example.com")).IsFalse()
	Expect(env.IsMCPOrigin("https://ideas.internal.example")).IsFalse()
	Expect(env.IsMCPOrigin("")).IsFalse()
}

func TestMCPOriginForHost(t *testing.T) {
	RegisterT(t)
	env.Config.MCPOrigins = []string{"https://mcp.example.com", "https://alt.example.com:8443"}
	defer func() { env.Config.MCPOrigins = nil }()
	for host, want := range map[string]string{
		"mcp.example.com":      "https://mcp.example.com",
		"MCP.Example.com:443":  "https://mcp.example.com",
		"alt.example.com:8443": "https://alt.example.com:8443",
	} {
		got, ok := env.MCPOriginForHost(host)
		Expect(ok).IsTrue()
		Expect(got).Equals(want)
	}
	// a trailing dot in Host is the same name
	got, ok := env.MCPOriginForHost("mcp.example.com.")
	Expect(ok).IsTrue()
	Expect(got).Equals("https://mcp.example.com")
	for _, host := range []string{"", "alt.example.com", "alt.example.com:443", "ideas.internal.example", "mcp.example.com.evil.com", "mcp.example.com:3000"} {
		_, ok := env.MCPOriginForHost(host)
		Expect(ok).IsFalse()
	}
}

func TestValidateMCPPort(t *testing.T) {
	RegisterT(t)
	origins := []string{"https://mcp.example.com"}
	Expect(env.ValidateMCPPort("", "3000", "", nil)).IsNil()
	Expect(env.ValidateMCPPort("3001", "3000", "4000", origins)).IsNil()
	Expect(env.ValidateMCPPort("3001", "3000", "", nil)).IsNotNil()     // a listener with no address to serve
	Expect(env.ValidateMCPPort("3000", "3000", "", origins)).IsNotNil() // the board's own port
	Expect(env.ValidateMCPPort("http", "3000", "", origins)).IsNotNil()
	Expect(env.ValidateMCPPort("70000", "3000", "", origins)).IsNotNil()
	Expect(env.ValidateMCPPort("4000", "3000", "4000", origins)).IsNotNil() // the metrics port
	Expect(env.ValidateMCPPort("03001", "3001", "", origins)).IsNotNil()    // same port, written differently
}
