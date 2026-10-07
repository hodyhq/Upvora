package jwt

import (
	"fmt"
	"time"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
	jwtgo "github.com/golang-jwt/jwt/v4"
)

var jwtSecret = env.Config.JWTSecret

// Metadata is the basic JWT information
type Metadata = jwtgo.RegisteredClaims

func Time(t time.Time) *jwtgo.NumericDate {
	return jwtgo.NewNumericDate(t)
}

const (
	//FiderClaimsOriginUI is assigned to Fider claims when the Auth Token is generated through the UI
	FiderClaimsOriginUI = "ui"
	//FiderClaimsOriginAPI is assigned to Fider claims when the Auth Token is generated through the API
	FiderClaimsOriginAPI = "api"
)

// FiderClaims represents what goes into JWT tokens
type FiderClaims struct {
	UserID        int    `json:"user/id"`
	UserName      string `json:"user/name"`
	UserEmail     string `json:"user/email"`
	Origin        string `json:"origin"`
	SecurityStamp string `json:"user/security_stamp,omitempty"`
	Metadata
}

// OAuthClaims represents what goes into temporary OAuth JWT tokens
type OAuthClaims struct {
	OAuthID       string `json:"oauth/id"`
	OAuthProvider string `json:"oauth/provider"`
	OAuthName     string `json:"oauth/name"`
	OAuthEmail    string `json:"oauth/email"`
	Metadata
}

// OAuthStateClaims represents what goes into JWT tokens used for OAuth state parameter
type OAuthStateClaims struct {
	Redirect   string `json:"oauthstate/redirect"`
	Identifier string `json:"oauthstate/identifier"`
	Code       string `json:"oauthstate/code"`
	Metadata
}

// OAuthHandoffClaims represents the short lived token that hands an OAuth sign in over
// from the host-wide callback address to the tenant address. It binds the handoff to a
// single provider, origin and browser session so it cannot be replayed elsewhere.
type OAuthHandoffClaims struct {
	Provider      string `json:"oauthhandoff/provider"`
	Origin        string `json:"oauthhandoff/origin"`
	SessionIDHash string `json:"oauthhandoff/session_id_hash"`
	Metadata
}

// MCPAccessClaims is an OAuth access token issued to an MCP client. It carries
// its own claim keys and always an audience (the site's /mcp URL), so it can
// never be decoded as a session token, and a session token (no audience) can
// never be decoded as one.
type MCPAccessClaims struct {
	UserID        int    `json:"mcp/user_id"`
	TenantID      int    `json:"mcp/tenant_id"`
	ClientID      string `json:"mcp/client_id"`
	Scope         string `json:"mcp/scope"`
	SecurityStamp string `json:"mcp/security_stamp"`
	Metadata
}

// Encode creates new JWT token with given claims
func Encode(claims jwtgo.Claims) (string, error) {
	jwtToken := jwtgo.NewWithClaims(jwtgo.GetSigningMethod("HS256"), claims)
	token, err := jwtToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", errors.Wrap(err, "failed to encode the requested claims")
	}
	return token, nil
}

// DecodeFiderClaims extract claims from JWT tokens
func DecodeFiderClaims(token string) (*FiderClaims, error) {
	claims := &FiderClaims{}
	err := decode(token, claims)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode Fider claims")
	}
	// Session tokens never carry an audience; anything that does is an access
	// token for another purpose (e.g. MCP) and is not a session.
	if len(claims.Audience) > 0 {
		return nil, errors.New("token with an audience is not a session token")
	}
	return claims, nil
}

// DecodeMCPAccessClaims decodes an MCP access token issued for audience.
func DecodeMCPAccessClaims(token, audience string) (*MCPAccessClaims, error) {
	claims := &MCPAccessClaims{}
	if err := decode(token, claims); err != nil {
		return nil, errors.Wrap(err, "failed to decode MCP access claims")
	}
	if audience == "" || !claims.VerifyAudience(audience, true) || len(claims.Audience) != 1 {
		return nil, errors.New("MCP access token audience mismatch")
	}
	if claims.UserID <= 0 || claims.TenantID <= 0 || claims.ExpiresAt == nil {
		return nil, errors.New("MCP access token is missing required claims")
	}
	return claims, nil
}

// DecodeOAuthClaims extract OAuthClaims from given JWT token
func DecodeOAuthClaims(token string) (*OAuthClaims, error) {
	claims := &OAuthClaims{}
	err := decode(token, claims)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode OAuth claims")
	}
	return claims, nil
}

// DecodeOAuthStateClaims extract OAuthClaims from given JWT token
func DecodeOAuthStateClaims(token string) (*OAuthStateClaims, error) {
	claims := &OAuthStateClaims{}
	err := decode(token, claims)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode OAuthState claims")
	}
	return claims, nil
}

// DecodeOAuthHandoffClaims extract OAuthHandoffClaims from given JWT token
func DecodeOAuthHandoffClaims(token string) (*OAuthHandoffClaims, error) {
	claims := &OAuthHandoffClaims{}
	err := decode(token, claims)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode OAuthHandoff claims")
	}
	return claims, nil
}

func decode(token string, claims jwtgo.Claims) error {
	jwtToken, err := jwtgo.ParseWithClaims(token, claims, func(t *jwtgo.Token) (any, error) {
		if _, ok := t.Method.(*jwtgo.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(jwtSecret), nil
	})

	if err == nil {
		err = claims.Valid()
	}

	if err == nil && jwtToken.Valid {
		return nil
	}

	return err
}
