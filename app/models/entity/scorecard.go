package entity

import (
	"encoding/json"
	"time"
)

// ScorecardField is one row on a tenant's scorecard schema — either a locked
// system-scoring dimension (weight + 1-5 slider) or an admin-added custom
// field (text/note/date/number/url/choice).
//
// Field types:
//   text | note | date | number | url | choice | score
//
// Groups (page sections on the card):
//   intake | context | workflow | ownership | classification | scoring | decision
type ScorecardField struct {
	ID         int             `json:"id"`
	Key        string          `json:"key"`
	Label      string          `json:"label"`
	GroupKey   string          `json:"groupKey"`
	Type       string          `json:"type"`
	Choices    json.RawMessage `json:"choices,omitempty"`
	Weight     *int            `json:"weight,omitempty"`
	Question   string          `json:"question,omitempty"`
	SortOrder  int             `json:"sortOrder"`
	IsSystem   bool            `json:"isSystem"`
	IsActive   bool            `json:"isActive"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// Scorecard is one committee-scored card, optionally linked to a Fider post.
// Field values live in a JSONB blob keyed by ScorecardField.Key.
type Scorecard struct {
	ID        int             `json:"id"`
	PostID    *int            `json:"postId,omitempty"`
	Title     string          `json:"title"`
	Values    json.RawMessage `json:"values"`
	CreatedBy *User           `json:"createdBy,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	// Post is populated when PostID != nil by handlers that stitch it on.
	// Not persisted; convenience payload for the card view.
	Post *Post `json:"post,omitempty"`
	// List-presentation fields, populated only by ListScorecardsForTenant's
	// post join so the dashboard shows live post data without N+1 fetches.
	PostNumber  *int   `json:"postNumber,omitempty"`
	PostSlug    string `json:"postSlug,omitempty"`
	PostVotes   int    `json:"postVotes,omitempty"`
	SubmittedBy string `json:"submittedBy,omitempty"`
}
