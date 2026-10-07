package cmd

import (
	"time"

	"github.com/getfider/fider/app/models/entity"
)

// RegisterOAuthClient creates a client with a random client_id.
type RegisterOAuthClient struct {
	Name           string
	RedirectURIs   []string
	CreatedByAdmin bool

	Result *entity.OAuthClient
}

// DeleteOAuthClient removes a client and revokes everything issued to it.
type DeleteOAuthClient struct {
	ID int
}

// SaveOAuthCode stores a hashed, single-use authorization code.
type SaveOAuthCode struct {
	CodeHash      string
	ClientID      string
	UserID        int
	RedirectURI   string
	Scope         string
	CodeChallenge string
	SecurityStamp string // user's stamp at issue; a later rotation invalidates the code
	Origin        string // MCP-only address it was issued on ('' on the board); only redeemable there
	ExpiresAt     time.Time
}

// ConsumeOAuthCode marks an unexpired, unused code for this client as used and
// returns its grant; app.ErrNotFound otherwise.
type ConsumeOAuthCode struct {
	CodeHash string
	ClientID string
	// Origin is the MCP-only address the exchange arrived on, or '' on the
	// board; a code bound elsewhere is refused (and not burned). AllowUnbound
	// accepts the board's own unbound codes: set it only off MCP-only addresses.
	Origin       string
	AllowUnbound bool

	Result *entity.OAuthGrant
}

// SaveOAuthRefreshToken stores a hashed refresh token in a rotation family.
type SaveOAuthRefreshToken struct {
	TokenHash string
	ClientID  string
	UserID    int
	Scope     string
	FamilyID  string
	// SecurityStamp is the user's stamp at issue; rotating it revokes the family.
	SecurityStamp string
	// FromCodeHash links the family to the code that started it, so a replay
	// of that code revokes it.
	FromCodeHash string
	Origin       string // MCP-only address it was issued on ('' on the board); only usable there
	ExpiresAt    time.Time
}

// RotateOAuthRefreshToken exchanges a live refresh token for a new one in the
// same family. Presenting a rotated or revoked token revokes the whole family
// and returns app.ErrNotFound.
type RotateOAuthRefreshToken struct {
	OldHash      string
	NewHash      string
	ClientID     string
	Origin       string // as ConsumeOAuthCode; a wrong address revokes nothing
	AllowUnbound bool
	NewExpiresAt time.Time

	Result *entity.OAuthGrant
}

// PurgeStaleOAuthData deletes old codes, dead refresh tokens and
// self-registered clients that were never used (all tenants).
type PurgeStaleOAuthData struct {
	Deleted int
}

// SaveOAuthSignInHandoff stores a hashed, single-use code that carries a
// signed-in person from the board's own address back to an MCP-only public
// address, together with the authorization request to resume there.
type SaveOAuthSignInHandoff struct {
	CodeHash      string
	UserID        int
	Origin        string
	Query         string
	SecurityStamp string
	// FlowHash is the hash of the flow cookie set on the MCP-only address
	// when the sign-in started: only that browser can redeem the code.
	FlowHash  string
	ExpiresAt time.Time
}

// RedeemOAuthSignInHandoff burns an unexpired handoff code issued for Origin
// whose user's security stamp is unchanged; app.ErrNotFound otherwise.
type RedeemOAuthSignInHandoff struct {
	CodeHash string
	Origin   string
	FlowHash string

	Result *entity.OAuthSignInHandoff
}
