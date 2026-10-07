package entity

import "time"

// OAuthClient is an MCP client registered with this site's authorization server.
type OAuthClient struct {
	ID             int       `json:"id"`
	ClientID       string    `json:"clientId"`
	Name           string    `json:"name"`
	RedirectURIs   []string  `json:"redirectUris"`
	CreatedByAdmin bool      `json:"createdByAdmin"`
	CreatedAt      time.Time `json:"createdAt"`
}

// OAuthGrant is what a consumed authorization code or rotated refresh token
// grants: who, to which client, with which scope.
type OAuthGrant struct {
	UserID        int
	ClientID      string
	Scope         string
	RedirectURI   string
	CodeChallenge string
	FamilyID      string
}

// OAuthSignInHandoff is a redeemed sign-in handoff: who signed in, and the
// authorization request to resume on the MCP-only address.
type OAuthSignInHandoff struct {
	UserID int
	Query  string
}
