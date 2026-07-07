package cmd

import (
	"encoding/json"

	"github.com/getfider/fider/app/models/entity"
)

// CreateScorecardField inserts a new admin-defined field on the tenant's
// scorecard schema. Score-type rows must land in the scoring group. Weight
// and Question are only meaningful when Type == "score".
type CreateScorecardField struct {
	Key       string
	Label     string
	GroupKey  string
	Type      string
	Choices   json.RawMessage
	Weight    *int
	Question  string
	SortOrder int

	Result *entity.ScorecardField
}

// UpdateScorecardField mutates an existing field. For is_system=true rows the
// caller must still supply Key/Type/GroupKey but the postgres layer ignores
// them (locked). Label, Weight, Question, Choices, SortOrder, IsActive are
// always writable.
type UpdateScorecardField struct {
	ID        int
	Label     string
	Choices   json.RawMessage
	Weight    *int
	Question  string
	SortOrder int
	IsActive  bool
}

// DeleteScorecardField removes an admin-added field. Refused for is_system rows.
// Card values that referenced the field key stay in the JSONB blob — harmless,
// just no longer rendered.
type DeleteScorecardField struct {
	ID int
}

// SeedTenantScorecardFields seeds the 8 locked scoring dimensions for a newly
// created tenant. Idempotent — ON CONFLICT (tenant_id, key) DO NOTHING.
type SeedTenantScorecardFields struct {
	TenantID int
}

// SetTenantScorecardSettings toggles the feature on/off and sets the four band
// thresholds in one shot. Called by the /admin/scorecard-settings page.
type SetTenantScorecardSettings struct {
	IsEnabled  bool
	BandStrong int
	BandGood   int
	BandRefine int
	BandLow    int
}
