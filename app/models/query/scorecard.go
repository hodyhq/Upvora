package query

import "github.com/getfider/fider/app/models/entity"

// ListScorecardFieldsForTenant returns every active scorecard field ordered by
// sort_order. Used at runtime — loaded onto tenant.ScorecardFields by middleware.
type ListScorecardFieldsForTenant struct {
	Result []*entity.ScorecardField
}

// ListAllScorecardFieldsForTenant returns every scorecard field including
// inactive ones. Used by the admin Manage Scorecard Fields page so admins
// can reactivate/edit fields they previously turned off.
type ListAllScorecardFieldsForTenant struct {
	Result []*entity.ScorecardField
}

// GetScorecardFieldByID fetches a single field from the current tenant by PK.
// Returns ErrNotFound when no row matches.
type GetScorecardFieldByID struct {
	ID     int
	Result *entity.ScorecardField
}

// ListScorecardsForTenant returns every scorecard for the current tenant,
// most recently updated first. Used by the /scorecard dashboard page.
type ListScorecardsForTenant struct {
	Result []*entity.Scorecard
}
