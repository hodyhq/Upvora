package actions

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/validate"
)

var scorecardKeyRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

var allowedScorecardFieldTypes = map[string]bool{
	"text": true, "note": true, "date": true, "number": true, "url": true, "choice": true, "score": true, "user": true,
}

var allowedScorecardGroups = map[string]bool{
	"intake": true, "context": true, "workflow": true, "ownership": true, "classification": true, "scoring": true, "decision": true,
}

// CreateScorecardField is the admin action to add a new field to the tenant's
// scorecard schema. The `score` type is reserved for the 8 seeded system
// fields; admins add customizable text/note/date/number/url/choice fields.
type CreateScorecardField struct {
	Key       string          `json:"key" format:"lower"`
	Label     string          `json:"label"`
	GroupKey  string          `json:"groupKey"`
	Type      string          `json:"type"`
	Choices   json.RawMessage `json:"choices,omitempty"`
	Weight    *int            `json:"weight,omitempty"`
	Question  string          `json:"question,omitempty"`
	SortOrder int             `json:"sortOrder"`
}

func (a *CreateScorecardField) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.Role == enum.RoleAdministrator
}

func (a *CreateScorecardField) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()

	a.Key = strings.TrimSpace(strings.ToLower(a.Key))
	a.Label = strings.TrimSpace(a.Label)
	a.GroupKey = strings.TrimSpace(a.GroupKey)
	a.Type = strings.TrimSpace(a.Type)

	if a.Key == "" {
		result.AddFieldFailure("key", "Key is required.")
	} else if len(a.Key) > 60 {
		result.AddFieldFailure("key", "Key must be 60 characters or fewer.")
	} else if !scorecardKeyRegex.MatchString(a.Key) {
		result.AddFieldFailure("key", "Key must use only lowercase letters, numbers, and underscores.")
	}

	if a.Label == "" {
		result.AddFieldFailure("label", "Label is required.")
	} else if len(a.Label) > 120 {
		result.AddFieldFailure("label", "Label must be 120 characters or fewer.")
	}

	if !allowedScorecardGroups[a.GroupKey] {
		result.AddFieldFailure("groupKey", "Group must be one of: intake, context, workflow, ownership, classification, scoring, decision.")
	}

	if !allowedScorecardFieldTypes[a.Type] {
		result.AddFieldFailure("type", "Type must be one of: text, note, date, number, url, choice, score.")
	}

	if a.Type == "score" && a.GroupKey != "scoring" {
		result.AddFieldFailure("groupKey", "Score fields must live in the Scoring group.")
	}

	if a.Type == "choice" && len(a.Choices) == 0 {
		result.AddFieldFailure("choices", "Choice fields must define a non-empty choices array.")
	}

	return result
}

// UpdateScorecardField is the admin action to edit an existing field. ID from
// the URL. Weight/Question are only meaningful when the field's type is
// `score` — for other types they should be null/empty.
type UpdateScorecardField struct {
	ID        int             `route:"id"`
	Label     string          `json:"label"`
	Choices   json.RawMessage `json:"choices,omitempty"`
	Weight    *int            `json:"weight,omitempty"`
	Question  string          `json:"question,omitempty"`
	SortOrder int             `json:"sortOrder"`
	IsActive  bool            `json:"isActive"`
}

func (a *UpdateScorecardField) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.Role == enum.RoleAdministrator
}

func (a *UpdateScorecardField) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	a.Label = strings.TrimSpace(a.Label)
	if a.Label == "" {
		result.AddFieldFailure("label", "Label is required.")
	} else if len(a.Label) > 120 {
		result.AddFieldFailure("label", "Label must be 120 characters or fewer.")
	}
	if a.Weight != nil && (*a.Weight < 0 || *a.Weight > 100) {
		result.AddFieldFailure("weight", "Weight must be between 0 and 100.")
	}
	return result
}

// DeleteScorecardField removes a non-system field. ID from the URL only.
type DeleteScorecardField struct {
	ID int `route:"id"`
}

func (a *DeleteScorecardField) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.Role == enum.RoleAdministrator
}

func (a *DeleteScorecardField) Validate(ctx context.Context, user *entity.User) *validate.Result {
	return validate.Success()
}

// CreateScorecard is the collaborator+ action to create a new scorecard,
// optionally linked to a Fider post. Idempotent per (tenant, post) — server
// returns the existing card if one already exists.
type CreateScorecard struct {
	PostID *int   `json:"postId,omitempty"`
	Title  string `json:"title"`
}

func (a *CreateScorecard) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && (user.Role == enum.RoleCollaborator || user.Role == enum.RoleAdministrator)
}

func (a *CreateScorecard) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	a.Title = strings.TrimSpace(a.Title)
	if len(a.Title) > 500 {
		result.AddFieldFailure("title", "Title must be 500 characters or fewer.")
	}
	return result
}

// UpdateScorecardValues writes the values JSON blob (and optional title).
type UpdateScorecardValues struct {
	ID     int             `route:"id"`
	Title  string          `json:"title"`
	Values json.RawMessage `json:"values"`
}

func (a *UpdateScorecardValues) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && (user.Role == enum.RoleCollaborator || user.Role == enum.RoleAdministrator)
}

func (a *UpdateScorecardValues) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	a.Title = strings.TrimSpace(a.Title)
	if a.Title == "" {
		result.AddFieldFailure("title", "Title is required.")
	} else if len(a.Title) > 500 {
		result.AddFieldFailure("title", "Title must be 500 characters or fewer.")
	}
	// values may be empty; JSON validity is verified downstream when Postgres
	// casts to JSONB (bad JSON returns a DB error which surfaces as 500 — good
	// enough for an admin-only surface).
	return result
}

// UpdateScorecardSettings toggles the feature, sets the 4 band thresholds,
// and picks the auto-create trigger status. Bands are 0-100 and must be
// strictly descending (strong > good > refine > low). TriggerStatusSlug is
// optional (empty = no auto-trigger).
type UpdateScorecardSettings struct {
	IsEnabled         bool   `json:"isEnabled"`
	BandStrong        int    `json:"bandStrong"`
	BandGood          int    `json:"bandGood"`
	BandRefine        int    `json:"bandRefine"`
	BandLow           int    `json:"bandLow"`
	TriggerStatusSlug string `json:"triggerStatusSlug"`
}

func (a *UpdateScorecardSettings) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.Role == enum.RoleAdministrator
}

func (a *UpdateScorecardSettings) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	for name, v := range map[string]int{"bandStrong": a.BandStrong, "bandGood": a.BandGood, "bandRefine": a.BandRefine, "bandLow": a.BandLow} {
		if v < 0 || v > 100 {
			result.AddFieldFailure(name, "Band threshold must be between 0 and 100.")
		}
	}
	if a.BandStrong <= a.BandGood || a.BandGood <= a.BandRefine || a.BandRefine <= a.BandLow {
		result.AddFieldFailure("bandStrong", "Band thresholds must be strictly descending: strong > good > refine > low.")
	}
	return result
}
