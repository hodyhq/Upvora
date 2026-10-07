package jwt_test

import (
	"testing"
	"time"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/jwt"
	jwtgo "github.com/golang-jwt/jwt/v4"
)

func TestJWT_Encode(t *testing.T) {
	RegisterT(t)

	claims := &jwt.FiderClaims{
		UserID:    424,
		UserName:  "Jon Snow",
		UserEmail: "jon.snow@got.com",
	}

	token, err := jwt.Encode(claims)
	Expect(token).IsNotEmpty()
	Expect(err).IsNil()
}

func TestJWT_Decode(t *testing.T) {
	RegisterT(t)

	claims := &jwt.FiderClaims{
		UserID:    424,
		UserName:  "Jon Snow",
		UserEmail: "jon.snow@got.com",
	}

	token, _ := jwt.Encode(claims)

	decoded, err := jwt.DecodeFiderClaims(token)
	Expect(err).IsNil()
	Expect(decoded.UserID).Equals(claims.UserID)
	Expect(decoded.UserName).Equals(claims.UserName)
	Expect(decoded.UserEmail).Equals(claims.UserEmail)
}

func TestJWT_Decode_DifferentSignMethod(t *testing.T) {
	RegisterT(t)

	jwtToken := jwtgo.NewWithClaims(jwtgo.GetSigningMethod("none"), &jwt.FiderClaims{
		UserID:    424,
		UserName:  "Jon Snow",
		UserEmail: "jon.snow@got.com",
	})
	token, err := jwtToken.SignedString(jwtgo.UnsafeAllowNoneSignatureType)
	Expect(err).IsNil()

	decoded, err := jwt.DecodeFiderClaims(token)
	Expect(err.Error()).ContainsSubstring("unexpected signing method: none")
	Expect(decoded).IsNil()
}

func TestJWT_DecodeExpired(t *testing.T) {
	RegisterT(t)

	claims := &jwt.FiderClaims{
		UserID:    424,
		UserName:  "Jon Snow",
		UserEmail: "jon.snow@got.com",
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(time.Now()),
		},
	}

	token, err := jwt.Encode(claims)
	Expect(err).IsNil()
	time.Sleep(1 * time.Second)

	decoded, err := jwt.DecodeFiderClaims(token)
	Expect(err).IsNotNil()
	Expect(decoded).IsNil()
}

func TestJWT_DecodeOAuthClaims(t *testing.T) {
	RegisterT(t)

	claims := &jwt.OAuthClaims{
		OAuthID:       "2",
		OAuthEmail:    "jon.snow@got.com",
		OAuthName:     "Jon Snow",
		OAuthProvider: "facebook",
	}

	token, _ := jwt.Encode(claims)

	decoded, err := jwt.DecodeOAuthClaims(token)
	Expect(err).IsNil()
	Expect(decoded.OAuthID).Equals(claims.OAuthID)
	Expect(decoded.OAuthEmail).Equals(claims.OAuthEmail)
	Expect(decoded.OAuthName).Equals(claims.OAuthName)
	Expect(decoded.OAuthProvider).Equals(claims.OAuthProvider)
}

func TestJWT_DecodeChangedToken(t *testing.T) {
	RegisterT(t)

	claims := &jwt.FiderClaims{
		UserID:    424,
		UserName:  "Jon Snow",
		UserEmail: "jon.snow@got.com",
	}

	token, _ := jwt.Encode(claims)

	decoded, err := jwt.DecodeFiderClaims(token + "foo")
	Expect(err).IsNotNil()
	Expect(decoded).IsNil()
}

func TestJWT_DecodeOAuthClaimsExpired(t *testing.T) {
	RegisterT(t)

	claims := &jwt.OAuthClaims{
		OAuthID:       "2",
		OAuthEmail:    "jon.snow@got.com",
		OAuthName:     "Jon Snow",
		OAuthProvider: "facebook",
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(time.Now()),
		},
	}

	token, err := jwt.Encode(claims)
	Expect(err).IsNil()
	time.Sleep(1 * time.Second)

	decoded, err := jwt.DecodeOAuthClaims(token)
	Expect(err).IsNotNil()
	Expect(decoded).IsNil()
}

func mcpClaims(aud string) *jwt.MCPAccessClaims {
	return &jwt.MCPAccessClaims{
		UserID: 7, TenantID: 1, ClientID: "c1", Scope: "upvora", SecurityStamp: "s",
		Metadata: jwt.Metadata{
			Audience:  jwtgo.ClaimStrings{aud},
			ExpiresAt: jwt.Time(time.Now().Add(time.Hour)),
		},
	}
}

func TestJWT_MCPAccess_RoundTripAndAudience(t *testing.T) {
	RegisterT(t)
	token, err := jwt.Encode(mcpClaims("https://demo.test/mcp"))
	Expect(err).IsNil()

	got, err := jwt.DecodeMCPAccessClaims(token, "https://demo.test/mcp")
	Expect(err).IsNil()
	Expect(got.UserID).Equals(7)
	Expect(got.Scope).Equals("upvora")

	_, err = jwt.DecodeMCPAccessClaims(token, "https://other.test/mcp")
	Expect(err).IsNotNil()
}

// A session token must never be accepted as an access token, and the reverse.
func TestJWT_MCPAccess_NoConfusionWithSessions(t *testing.T) {
	RegisterT(t)
	session, _ := jwt.Encode(&jwt.FiderClaims{UserID: 7, Metadata: jwt.Metadata{ExpiresAt: jwt.Time(time.Now().Add(time.Hour))}})
	_, err := jwt.DecodeMCPAccessClaims(session, "https://demo.test/mcp")
	Expect(err).IsNotNil()

	access, _ := jwt.Encode(mcpClaims("https://demo.test/mcp"))
	_, err = jwt.DecodeFiderClaims(access)
	Expect(err).IsNotNil()
}

func TestJWT_MCPAccess_Expired(t *testing.T) {
	RegisterT(t)
	c := mcpClaims("https://demo.test/mcp")
	c.ExpiresAt = jwt.Time(time.Now().Add(-time.Minute))
	token, _ := jwt.Encode(c)
	_, err := jwt.DecodeMCPAccessClaims(token, "https://demo.test/mcp")
	Expect(err).IsNotNil()
}

func TestJWT_MCPSignInRequest_RoundTripAndIsolation(t *testing.T) {
	RegisterT(t)
	token, err := jwt.EncodeMCPSignInRequest(1, "https://mcp.demo.test", "client_id=c&state=s", "flowhash", time.Minute)
	Expect(err).IsNil()
	claims, err := jwt.DecodeMCPSignInRequest(token)
	Expect(err).IsNil()
	Expect(claims.TenantID).Equals(1)
	Expect(claims.Origin).Equals("https://mcp.demo.test")
	Expect(claims.Query).Equals("client_id=c&state=s")
	Expect(claims.FlowHash).Equals("flowhash")

	// never a session, an MCP access token or a consent token
	_, err = jwt.DecodeFiderClaims(token)
	Expect(err).IsNotNil()
	_, err = jwt.DecodeMCPAccessClaims(token, "https://mcp.demo.test/mcp")
	Expect(err).IsNotNil()
	_, err = jwt.DecodeMCPConsent(token, "https://mcp.demo.test")
	Expect(err).IsNotNil()

	expired, _ := jwt.EncodeMCPSignInRequest(1, "https://mcp.demo.test", "q", "f", -time.Minute)
	_, err = jwt.DecodeMCPSignInRequest(expired)
	Expect(err).IsNotNil()
}

func TestJWT_MCPConsent_BoundToOrigin(t *testing.T) {
	RegisterT(t)
	token, err := jwt.EncodeMCPConsent(7, 1, "stamp", "https://mcp.demo.test", time.Minute)
	Expect(err).IsNil()
	claims, err := jwt.DecodeMCPConsent(token, "https://mcp.demo.test")
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(7)
	Expect(claims.TenantID).Equals(1)
	Expect(claims.SecurityStamp).Equals("stamp")

	_, err = jwt.DecodeMCPConsent(token, "https://other.demo.test")
	Expect(err).IsNotNil()
	_, err = jwt.DecodeFiderClaims(token)
	Expect(err).IsNotNil()
	_, err = jwt.DecodeMCPAccessClaims(token, "https://mcp.demo.test/mcp")
	Expect(err).IsNotNil()
	_, err = jwt.DecodeMCPSignInRequest(token)
	Expect(err).IsNotNil()
}
