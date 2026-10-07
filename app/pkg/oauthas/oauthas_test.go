package oauthas_test

import (
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/oauthas"
)

func TestNewToken(t *testing.T) {
	RegisterT(t)
	a, hashA := oauthas.NewToken()
	b, _ := oauthas.NewToken()
	Expect(len(a) >= 43).IsTrue() // 32 random bytes, base64url
	Expect(a == b).IsFalse()
	Expect(hashA).Equals(oauthas.HashToken(a))
	Expect(len(hashA)).Equals(64)
}

// RFC 7636 Appendix B test vector.
func TestVerifyPKCE(t *testing.T) {
	RegisterT(t)
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	Expect(oauthas.VerifyPKCE(verifier, challenge, "S256")).IsTrue()
	Expect(oauthas.VerifyPKCE(verifier+"x", challenge, "S256")).IsFalse()
	Expect(oauthas.VerifyPKCE(verifier, verifier, "plain")).IsFalse()
	Expect(oauthas.VerifyPKCE("short", "short", "S256")).IsFalse()
	Expect(oauthas.VerifyPKCE("", challenge, "S256")).IsFalse()
}

func TestValidRedirectURI(t *testing.T) {
	RegisterT(t)
	for _, ok := range []string{
		"https://claude.ai/api/mcp/auth_callback",
		"http://127.0.0.1:33418/callback",
		"http://localhost:6274/oauth/callback",
		"http://[::1]:8080/cb",
		"cursor://anysphere.cursor-retrieval/oauth/callback",
	} {
		Expect(oauthas.ValidRedirectURI(ok)).IsTrue()
	}
	for _, bad := range []string{
		"", "not a url", "/relative/path",
		"http://evil.example/callback",       // plain http to a remote host
		"https://claude.ai/cb#fragment",       // fragments are forbidden
		"javascript:alert(1)", "data:text/html,x", "vbscript:x", "file:///etc/passwd",
		"https://user:pass@claude.ai/cb",      // userinfo
	} {
		Expect(oauthas.ValidRedirectURI(bad)).IsFalse()
	}
}

func TestParseScope(t *testing.T) {
	RegisterT(t)
	cases := map[string]string{"": "upvora", "upvora": "upvora", "upvora:read": "upvora:read", "upvora upvora:read": "upvora"}
	for in, want := range cases {
		got, ok := oauthas.ParseScope(in)
		Expect(ok).IsTrue()
		Expect(got).Equals(want)
	}
	_, ok := oauthas.ParseScope("admin")
	Expect(ok).IsFalse()
}
