package query

import "github.com/getfider/fider/app/models/entity"

// ListScorecardFieldsForTenant returns every active scorecard field ordered by
// sort_order. Used both by the admin field-management UI and by the runtime
// card view. Loaded onto tenant.ScorecardFields by middleware.
type ListScorecardFieldsForTenant struct {
	Result []*entity.ScorecardField
}

// GetScorecardFieldByID fetches a single field from the current tenant by PK.
// Returns ErrNotFound when no row matches.
type GetScorecardFieldByID struct {
	ID     int
	Result *entity.ScorecardField
}
