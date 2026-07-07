package handlers

import (
	"net/http"

	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// ScorecardPage renders the collaborator-facing dashboard of scorecards.
// Route is behind the collaborator+ middleware. When the feature is disabled
// on the tenant, still render (empty state) — the header link is what hides.
func ScorecardPage() web.HandlerFunc {
	return func(c *web.Context) error {
		list := &query.ListScorecardsForTenant{}
		if err := bus.Dispatch(c, list); err != nil {
			return c.Failure(err)
		}
		return c.Page(http.StatusOK, web.Props{
			Page:  "Scorecard/Scorecard.page",
			Title: "Scorecard",
			Data: web.Map{
				"scorecards": list.Result,
			},
		})
	}
}
