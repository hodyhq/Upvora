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
	ExpiresAt time.Time
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
