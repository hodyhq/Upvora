package handlers

import (
	"net/http"

	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// ManageScorecardSettings renders the merged Scorecard admin page: the feature
// toggle + band stages AND the field catalogue (previously its own page).
func ManageScorecardSettings() web.HandlerFunc {
	return func(c *web.Context) error {
		list := &query.ListAllScorecardFieldsForTenant{}
		if err := bus.Dispatch(c, list); err != nil {
			return c.Failure(err)
		}
		// Per-key answer counts drive delete-vs-deactivate gating client-side.
		// Best-effort: an empty map just means every field shows Delete.
		usage := &query.GetScorecardFieldUsage{}
		if err := bus.Dispatch(c, usage); err != nil {
			usage.Result = map[string]int{}
		}
		return c.Page(http.StatusOK, web.Props{
			Page:  "Administration/pages/ScorecardSettings.page",
			Title: "Scorecard · Site Settings",
			Data: web.Map{
				"fields": list.Result,
				"usage":  usage.Result,
			},
		})
	}
}

// LegacyScorecardFieldsRedirect sends the retired /admin/scorecard-fields URL
// to the merged Scorecard page.
func LegacyScorecardFieldsRedirect() web.HandlerFunc {
	return func(c *web.Context) error {
		return c.Redirect("/admin/scorecard-settings")
	}
}
