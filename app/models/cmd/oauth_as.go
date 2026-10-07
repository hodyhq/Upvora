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
	ExpiresAt     time.Time
}

// ConsumeOAuthCode marks an unexpired, unused code for this client as used and
// returns its grant; app.ErrNotFound otherwise.
type ConsumeOAuthCode struct {
	CodeHash string
	ClientID string

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
	ExpiresAt    time.Time
}

// RotateOAuthRefreshToken exchanges a live refresh token for a new one in the
// same family. Presenting a rotated or revoked token revokes the whole family
// and returns app.ErrNotFound.
type RotateOAuthRefreshToken struct {
	OldHash      string
	NewHash      string
	ClientID     string
	NewExpiresAt time.Time

	Result *entity.OAuthGrant
}
