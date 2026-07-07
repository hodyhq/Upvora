package handlers

import (
	"net/http"

	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// ManageScorecardFields renders the admin page where tenants configure their
// scorecard field catalogue (add custom fields, edit weights/questions on the
// eight seeded scoring dimensions, reorder, deactivate).
func ManageScorecardFields() web.HandlerFunc {
	return func(c *web.Context) error {
		list := &query.ListAllScorecardFieldsForTenant{}
		if err := bus.Dispatch(c, list); err != nil {
			return c.Failure(err)
		}
		return c.Page(http.StatusOK, web.Props{
			Page:  "Administration/pages/ManageScorecardFields.page",
			Title: "Scorecard Fields · Site Settings",
			Data: web.Map{
				"fields": list.Result,
			},
		})
	}
}
