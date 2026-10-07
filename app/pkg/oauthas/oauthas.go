// Package oauthas holds the primitives of Upvora's embedded OAuth 2.1
// authorization server (used by MCP clients): opaque tokens, PKCE, redirect
// URI and scope rules.
package oauthas

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// ScopeCtxKey holds the scope of the MCP access token on the request context.
type ScopeCtxKey struct{}

const (
	// ScopeFull acts as the user with UI-equivalent permissions.
	ScopeFull = "upvora"
	// ScopeRead is read-only: any non-GET request is refused.
	ScopeRead = "upvora:read"
)

// NewToken returns a new opaque token (32 bytes from crypto/rand, base64url)
// and the hash to store. Only the hash is ever persisted.
func NewToken() (plain, hash string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain)
}

// HashToken is the storage form of a high-entropy token (SHA-256, hex). A slow
// KDF is unnecessary: the input has 256 bits of entropy.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

var verifierRE = regexp.MustCompile(`^[A-Za-z0-9\-._~]{43,128}$`)

// VerifyPKCE checks an RFC 7636 S256 code verifier against its challenge.
// The "plain" method is not supported (OAuth 2.1).
func VerifyPKCE(verifier, challenge, method string) bool {
	if method != "S256" || !verifierRE.MatchString(verifier) || challenge == "" {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// ValidChallenge reports whether an S256 code_challenge is well formed.
func ValidChallenge(challenge string) bool {
	return len(challenge) == 43 && verifierRE.MatchString(challenge)
}

var blockedSchemes = map[string]bool{
	"javascript": true, "data": true, "vbscript": true, "file": true, "blob": true, "about": true,
}

// ValidRedirectURI applies the registration rules for redirect URIs: https;
// http only to a loopback host (RFC 8252); or a native-app private-use scheme.
// Never a fragment, userinfo, or a scheme a browser would execute.
func ValidRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Fragment != "" || strings.Contains(raw, "#") || u.User != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	switch {
	case blockedSchemes[scheme]:
		return false
	case scheme == "https":
		return u.Host != ""
	case scheme == "http":
		host := u.Hostname()
		if host == "localhost" {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	default:
		// private-use scheme for a native app, e.g. cursor://...
		return u.Host != "" || u.Opaque != "" || u.Path != ""
	}
}

// ParseScope normalizes a requested scope. Empty means full access (the
// consent screen shows it); any unknown scope is rejected.
func ParseScope(s string) (string, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ScopeFull, true
	}
	full := false
	for _, f := range fields {
		switch f {
		case ScopeFull:
			full = true
		case ScopeRead:
		default:
			return "", false
		}
	}
	if full {
		return ScopeFull, true
	}
	return ScopeRead, true
}
