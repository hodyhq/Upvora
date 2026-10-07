package apiv1_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers/apiv1"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jsonq"
	"github.com/getfider/fider/app/pkg/mock"
)

// tenantWithSecrets is the demo tenant carrying every secret a settings read
// must never return.
func tenantWithSecrets() *entity.Tenant {
	t := *mock.DemoTenant
	t.AIAPIKey = "sk-provider-secret"
	t.AIWebSearchAPIKey = "ws-search-secret"
	t.DeletionCancelKey = "cancel-key-secret"
	t.CustomCSS = ".brand { color: red }"
	return &t
}

func noSecrets(body string) {
	for _, secret := range []string{"sk-provider-secret", "ws-search-secret", "cancel-key-secret"} {
		Expect(strings.Contains(body, secret)).IsFalse()
	}
}

func TestGetTenantHandler(t *testing.T) {
	RegisterT(t)

	code, res := mock.NewServer().OnTenant(tenantWithSecrets()).Execute(apiv1.GetTenant())

	Expect(code).Equals(http.StatusOK)
	Expect(res.Body.String()).ContainsSubstring(`"membersPrivateIdeas"`)
	noSecrets(res.Body.String())
}

func TestGetAdvancedSettingsHandler(t *testing.T) {
	RegisterT(t)

	code, res := mock.NewServer().OnTenant(tenantWithSecrets()).AsUser(mock.JonSnow).Execute(apiv1.GetAdvancedSettings())

	Expect(code).Equals(http.StatusOK)
	Expect(res.Body.String()).ContainsSubstring(".brand { color: red }")
	noSecrets(res.Body.String())
}

func TestGetAISettingsHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.ListAIAgents) error {
		q.Result = []*entity.AIAgent{{Instructions: "be kind"}}
		return nil
	})

	code, res := mock.NewServer().OnTenant(tenantWithSecrets()).AsUser(mock.JonSnow).Execute(apiv1.GetAISettings())

	Expect(code).Equals(http.StatusOK)
	Expect(res.Body.String()).ContainsSubstring(`"hasKey":true`)
	Expect(res.Body.String()).ContainsSubstring(`"webSearchHasKey":true`)
	Expect(res.Body.String()).ContainsSubstring("be kind")
	noSecrets(res.Body.String())
}

func TestListWebhooksHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.ListAllWebhooks) error {
		q.Result = []*entity.Webhook{{ID: 7, Name: "Slack"}}
		return nil
	})

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).ExecuteAsJSON(apiv1.ListWebhooks())

	Expect(code).Equals(http.StatusOK)
	Expect(res.ArrayLength()).Equals(1)
}

func TestListOAuthProvidersHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.ListAllOAuthProviders) error {
		q.Result = []*dto.OAuthProviderOption{{Provider: "_custom"}}
		return nil
	})

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).ExecuteAsJSON(apiv1.ListOAuthProviders())

	Expect(code).Equals(http.StatusOK)
	Expect(res.ArrayLength()).Equals(1)
}

func TestListScorecardsHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.ListScorecardsForTenant) error {
		q.Result = []*entity.Scorecard{{ID: 3}}
		return nil
	})

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).ExecuteAsJSON(apiv1.ListScorecards())

	Expect(code).Equals(http.StatusOK)
	Expect(res.ArrayLength()).Equals(1)
}

func TestGetScorecardHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.GetScorecardByID) error {
		if q.ID != 3 {
			return app.ErrNotFound
		}
		q.Result = &entity.Scorecard{ID: 3}
		return nil
	})

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).AddParam("id", 3).ExecuteAsJSON(apiv1.GetScorecard())
	Expect(code).Equals(http.StatusOK)
	Expect(res.Int32("id")).Equals(3)

	code, _ = mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).AddParam("id", 99).ExecuteAsJSON(apiv1.GetScorecard())
	Expect(code).Equals(http.StatusNotFound)
}

func TestGetScorecardSettingsHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.ListAllScorecardFieldsForTenant) error {
		q.Result = []*entity.ScorecardField{{Key: "impact"}}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetScorecardFieldUsage) error {
		q.Result = map[string]int{"impact": 2}
		return nil
	})

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).ExecuteAsJSON(apiv1.GetScorecardSettings())

	Expect(code).Equals(http.StatusOK)
	Expect(jsonq.New(string(res.Raw("fields"))).ArrayLength()).Equals(1)
	Expect(res.Int32("usage.impact")).Equals(2)
}

func TestGetBillingStateHandler(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.GetStripeBillingState) error {
		q.Result = &entity.StripeBillingState{}
		return nil
	})

	code, _ := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).ExecuteAsJSON(apiv1.GetBillingState())

	Expect(code).Equals(http.StatusOK)
}
