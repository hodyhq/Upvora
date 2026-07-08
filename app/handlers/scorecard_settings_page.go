package handlers

import (
	"net/http"

	"github.com/getfider/fider/app/pkg/web"
)

// ManageScorecardSettings renders the admin page where tenants toggle the
// scorecard feature and set the four band thresholds. The band defaults and
// current values come off Fider.session.tenant on the client.
func ManageScorecardSettings() web.HandlerFunc {
	return func(c *web.Context) error {
		return c.Page(http.StatusOK, web.Props{
			Page:  "Administration/pages/ScorecardSettings.page",
			Title: "Scorecard · Site Settings",
		})
	}
}
