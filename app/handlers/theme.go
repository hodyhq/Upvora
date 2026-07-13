package handlers

import (
	"net/http"

	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// ManageThemePage renders the Admin - Theme page. Current values ride in on
// Fider.session.tenant client-side; no extra data needed.
func ManageThemePage() web.HandlerFunc {
	return func(c *web.Context) error {
		return c.Page(http.StatusOK, web.Props{
			Page:  "Administration/pages/ManageTheme.page",
			Title: "Theme · Site Settings",
		})
	}
}

// UpdateTenantTheme persists the token-based theme.
func UpdateTenantTheme() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.UpdateTenantTheme)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		set := &cmd.SetTenantTheme{
			Primary:      action.Primary,
			Accents:      action.Accents,
			DefaultTheme: action.DefaultTheme,
		}
		if err := bus.Dispatch(c, set); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}
