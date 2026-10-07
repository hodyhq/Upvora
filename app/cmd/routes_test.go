package cmd

import (
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/web"
)

func TestGetMainEngine(t *testing.T) {
	RegisterT(t)

	r := routes(web.New())
	Expect(r).IsNotNil()
}

// apiV1Surface is the full Bearer-authed API. Every UI action and settings read
// is reachable here (Phase 1 of v0.38.2.7.0); keeping existing routes listed
// stops one from silently disappearing.
var apiV1Surface = []string{
	// public
	"GET /api/v1/posts", "GET /api/v1/posts/:number", "GET /api/v1/posts/:number/comments",
	"GET /api/v1/posts/:number/comments/:id", "GET /api/v1/posts/:number/votes",
	"GET /api/v1/similarposts", "GET /api/v1/tags", "GET /api/v1/taggable-users",
	"GET /api/v1/tenant",

	// member
	"POST /api/v1/posts", "PUT /api/v1/posts/:number",
	"POST /api/v1/posts/:number/comments", "PUT /api/v1/posts/:number/comments/:id", "DELETE /api/v1/posts/:number/comments/:id",
	"POST /api/v1/posts/:number/comments/:id/reactions/:reaction",
	"POST /api/v1/posts/:number/votes", "DELETE /api/v1/posts/:number/votes", "POST /api/v1/posts/:number/votes/toggle",
	"POST /api/v1/posts/:number/subscription", "DELETE /api/v1/posts/:number/subscription",
	"POST /api/v1/ai/ideate", "POST /api/v1/ai/finalize", "GET /api/v1/posts/:number/brief",
	"GET /api/v1/notifications", "GET /api/v1/notifications/unread/total", "POST /api/v1/notifications/read-all",
	"POST /api/v1/user/settings", "POST /api/v1/user/change-email",
	"POST /api/v1/posts/:number/privacy",

	// collaborator
	"PUT /api/v1/posts/:number/status",
	"GET /api/v1/users", "POST /api/v1/invitations/send", "POST /api/v1/invitations/sample",
	"POST /api/v1/posts/:number/tags/:slug", "DELETE /api/v1/posts/:number/tags/:slug",
	"POST /api/v1/tags", "PUT /api/v1/tags/:slug", "DELETE /api/v1/tags/:slug",
	"PUT /api/v1/posts/:number/product",
	"GET /api/v1/posts/:number/internal-note", "PUT /api/v1/posts/:number/internal-note",
	"GET /api/v1/scorecards", "POST /api/v1/scorecards",
	"GET /api/v1/scorecards/:id", "PUT /api/v1/scorecards/:id", "DELETE /api/v1/scorecards/:id",
	"GET /api/v1/admin/scorecard-fields", "POST /api/v1/admin/scorecard-fields",
	"PUT /api/v1/admin/scorecard-fields/:id", "DELETE /api/v1/admin/scorecard-fields/:id",
	"GET /api/v1/admin/scorecard-settings", "POST /api/v1/admin/scorecard-settings",
	"POST /api/v1/admin/settings/theme",
	"GET /api/v1/admin/products", "POST /api/v1/admin/products",
	"PUT /api/v1/admin/products/:id", "DELETE /api/v1/admin/products/:id",

	// admin
	"POST /api/v1/users", "DELETE /api/v1/posts/:number",
	"GET /api/v1/admin/settings/advanced",
	"GET /api/v1/admin/settings/ai", "POST /api/v1/admin/settings/ai", "POST /api/v1/admin/ai/agents",
	"GET /api/v1/posts/:number/brief/transcript",
	"GET /api/v1/admin/system/status", "POST /api/v1/admin/system/update",
	"POST /api/v1/admin/settings/general", "POST /api/v1/admin/settings/advanced", "POST /api/v1/admin/settings/privacy",
	"POST /api/v1/admin/settings/emailauth", "POST /api/v1/admin/settings/site-banner",
	"GET /api/v1/admin/statuses", "POST /api/v1/admin/statuses",
	"PUT /api/v1/admin/statuses/:id", "DELETE /api/v1/admin/statuses/:id",
	"GET /api/v1/admin/webhooks", "POST /api/v1/admin/webhooks",
	"PUT /api/v1/admin/webhooks/:id", "DELETE /api/v1/admin/webhooks/:id",
	"POST /api/v1/admin/webhooks/test/:id", "POST /api/v1/admin/webhooks/preview", "GET /api/v1/admin/webhooks/props/:type",
	"POST /api/v1/admin/import/tags",
	"GET /api/v1/admin/oauth", "POST /api/v1/admin/oauth",
	"GET /api/v1/admin/oauth/:provider", "POST /api/v1/admin/oauth/:provider/status",
	"POST /api/v1/admin/roles/:role/users",
	"PUT /api/v1/admin/users/:userID/block", "DELETE /api/v1/admin/users/:userID/block",
	"PUT /api/v1/admin/users/:userID/trust", "DELETE /api/v1/admin/users/:userID/trust",

	// admin + pro
	"GET /api/v1/admin/moderation/items", "GET /api/v1/admin/moderation/count",
}

func TestAPIv1Surface(t *testing.T) {
	RegisterT(t)

	registered := map[string]bool{}
	for _, r := range routes(web.New()).Routes() {
		registered[r] = true
	}

	expected := append([]string{}, apiV1Surface...)
	if env.IsBillingEnabled() {
		expected = append(expected, "GET /api/v1/admin/billing", "POST /api/v1/admin/billing/portal",
			"POST /api/v1/admin/billing/checkout", "POST /api/v1/admin/billing/checkout/annual")
	}
	if !env.IsSingleHostMode() {
		expected = append(expected, "DELETE /api/v1/admin/tenant", "POST /api/v1/admin/tenant/cancel-deletion")
	}

	missing := []string{}
	for _, r := range expected {
		if !registered[r] {
			missing = append(missing, r)
		}
	}
	Expect(missing).Equals([]string{})
}
