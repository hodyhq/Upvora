package apiv1

import (
	"net/url"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// Settings and resource reads for the API. Each mirrors the data its admin page
// loads, so API and MCP clients can read state before writing it. Settings
// writes take the full settings object: read, modify, write.

// GetTenant returns the tenant as every page already receives it (secrets are
// json:"-" on the entity).
func GetTenant() web.HandlerFunc {
	return func(c *web.Context) error {
		return c.Ok(c.Tenant())
	}
}

// GetAdvancedSettings returns the Advanced tab's fields (CustomCSS is not in
// the tenant JSON).
func GetAdvancedSettings() web.HandlerFunc {
	return func(c *web.Context) error {
		tenant := c.Tenant()
		return c.Ok(web.Map{
			"customCSS":      tenant.CustomCSS,
			"allowedSchemes": tenant.AllowedSchemes,
		})
	}
}

// GetAISettings returns the AI tab's data. Provider keys are reported only as
// hasKey booleans, exactly as the admin page does.
func GetAISettings() web.HandlerFunc {
	return func(c *web.Context) error {
		agents := &query.ListAIAgents{}
		if err := bus.Dispatch(c, agents); err != nil {
			return c.Failure(err)
		}
		tenant := c.Tenant()
		return c.Ok(web.Map{
			"agents":            agents.Result,
			"enabled":           tenant.AIEnabled,
			"provider":          tenant.AIProvider,
			"model":             tenant.AIModel,
			"customBaseUrl":     tenant.AICustomBaseURL,
			"customModel":       tenant.AICustomModel,
			"hasKey":            tenant.AIAPIKey != "",
			"webSearchEnabled":  tenant.AIWebSearchEnabled,
			"webSearchProvider": tenant.AIWebSearchProvider,
			"webSearchBaseUrl":  tenant.AIWebSearchBaseURL,
			"webSearchHasKey":   tenant.AIWebSearchAPIKey != "",
		})
	}
}

// ListWebhooks returns every webhook with its secrets masked. Webhook URLs
// (Slack, Discord) and headers usually carry credentials; API and MCP clients
// get scheme and host only and masked header values. The admin page keeps the
// full values for editing.
func ListWebhooks() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.ListAllWebhooks{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		masked := make([]*entity.Webhook, len(q.Result))
		for i, w := range q.Result {
			m := *w
			m.Url = maskURL(w.Url)
			m.HttpHeaders = entity.HttpHeaders{}
			for name := range w.HttpHeaders {
				m.HttpHeaders[name] = "***"
			}
			masked[i] = &m
		}
		return c.Ok(masked)
	}
}

func maskURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "***"
	}
	masked := u.Scheme + "://" + u.Host + "/"
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" {
		masked += "***"
	}
	return masked
}

// ListOAuthProviders returns the configured sign-in providers (no secrets).
func ListOAuthProviders() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.ListAllOAuthProviders{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// ListScorecards returns every scorecard in the tenant.
func ListScorecards() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.ListScorecardsForTenant{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// GetScorecard returns one scorecard.
func GetScorecard() web.HandlerFunc {
	return func(c *web.Context) error {
		id, err := c.ParamAsInt("id")
		if err != nil {
			return c.NotFound()
		}
		q := &query.GetScorecardByID{ID: id}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// GetScorecardSettings returns every scorecard field (inactive too) with its
// usage count. Band settings ride on the tenant (GET /api/v1/tenant).
func GetScorecardSettings() web.HandlerFunc {
	return func(c *web.Context) error {
		fields := &query.ListAllScorecardFieldsForTenant{}
		if err := bus.Dispatch(c, fields); err != nil {
			return c.Failure(err)
		}
		usage := &query.GetScorecardFieldUsage{}
		if err := bus.Dispatch(c, usage); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"fields": fields.Result, "usage": usage.Result})
	}
}

// GetBillingState returns the tenant's subscription state.
func GetBillingState() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.GetStripeBillingState{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// GetUserSettings returns the signed-in user's editable settings, the same
// fields POST /api/v1/user/settings accepts.
func GetUserSettings() web.HandlerFunc {
	return func(c *web.Context) error {
		settings := &query.GetCurrentUserSettings{}
		if err := bus.Dispatch(c, settings); err != nil {
			return c.Failure(err)
		}
		user := c.User()
		return c.Ok(web.Map{
			"name":       user.Name,
			"avatarType": user.AvatarType,
			"settings":   settings.Result,
		})
	}
}
