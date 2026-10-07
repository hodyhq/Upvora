package jwt

import (
	"time"

	jwtgo "github.com/golang-jwt/jwt/v4"

	"github.com/getfider/fider/app/pkg/errors"
)

// Signing in for an MCP-only public address (MCP_ORIGINS) happens on the
// board's own address. Two short-lived tokens carry the flow across; each has
// its own claim keys and a fixed audience, so neither can be decoded as a
// session, an MCP access token or the other.

const mcpSignInAudience = "upvora:mcp-signin"

func mcpConsentAudience(origin string) string { return "upvora:mcp-consent:" + origin }

// MCPSignInRequestClaims carries an authorization request from an MCP-only
// address to the board's sign-in, and names the address to return to.
type MCPSignInRequestClaims struct {
	TenantID int    `json:"mcpsignin/tenant_id"`
	Origin   string `json:"mcpsignin/origin"`
	Query    string `json:"mcpsignin/query"`
	// FlowHash is the hash of the flow cookie set in the browser that started
	// the sign-in on the MCP-only address; only that browser can finish it.
	FlowHash string `json:"mcpsignin/flow_hash"`
	Metadata
}

// MCPConsentClaims identifies a person on an MCP-only address for the consent
// step only (the consent page and its decision), after signing in on the board.
type MCPConsentClaims struct {
	UserID        int    `json:"mcpconsent/user_id"`
	TenantID      int    `json:"mcpconsent/tenant_id"`
	SecurityStamp string `json:"mcpconsent/security_stamp"`
	Metadata
}

func shortLived(audience string, ttl time.Duration) Metadata {
	now := time.Now()
	return Metadata{
		Audience:  jwtgo.ClaimStrings{audience},
		IssuedAt:  Time(now),
		ExpiresAt: Time(now.Add(ttl)),
	}
}

// EncodeMCPSignInRequest signs a sign-in request valid for ttl.
func EncodeMCPSignInRequest(tenantID int, origin, query, flowHash string, ttl time.Duration) (string, error) {
	return Encode(&MCPSignInRequestClaims{TenantID: tenantID, Origin: origin, Query: query, FlowHash: flowHash, Metadata: shortLived(mcpSignInAudience, ttl)})
}

// DecodeMCPSignInRequest decodes an unexpired sign-in request.
func DecodeMCPSignInRequest(token string) (*MCPSignInRequestClaims, error) {
	claims := &MCPSignInRequestClaims{}
	if err := decode(token, claims); err != nil {
		return nil, errors.Wrap(err, "failed to decode MCP sign-in request")
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != mcpSignInAudience || claims.ExpiresAt == nil ||
		claims.TenantID <= 0 || claims.Origin == "" || claims.FlowHash == "" {
		return nil, errors.New("not an MCP sign-in request")
	}
	return claims, nil
}

// EncodeMCPConsent signs a consent-step identity for origin, valid for ttl.
func EncodeMCPConsent(userID, tenantID int, securityStamp, origin string, ttl time.Duration) (string, error) {
	return Encode(&MCPConsentClaims{UserID: userID, TenantID: tenantID, SecurityStamp: securityStamp, Metadata: shortLived(mcpConsentAudience(origin), ttl)})
}

// DecodeMCPConsent decodes an unexpired consent-step identity for origin.
func DecodeMCPConsent(token, origin string) (*MCPConsentClaims, error) {
	claims := &MCPConsentClaims{}
	if err := decode(token, claims); err != nil {
		return nil, errors.Wrap(err, "failed to decode MCP consent")
	}
	if origin == "" || len(claims.Audience) != 1 || claims.Audience[0] != mcpConsentAudience(origin) || claims.ExpiresAt == nil ||
		claims.UserID <= 0 || claims.TenantID <= 0 {
		return nil, errors.New("not an MCP consent token for this address")
	}
	return claims, nil
}
