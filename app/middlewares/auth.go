package middlewares

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/web"
)

// IsAuthenticated blocks non-authenticated requests. Page navigations get a
// sign-in redirect back to where they were headed (a signed-out admin hitting
// /scorecard or /admin should land on sign-in, not a dead 401); API and AJAX
// calls keep the raw 401.
func IsAuthenticated() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if !c.IsAuthenticated() {
				path := c.Request.URL.Path
				isAPI := strings.HasPrefix(path, "/_api") || strings.HasPrefix(path, "/api/")
				if c.Request.Method == http.MethodGet && !c.IsAjax() && !isAPI {
					return c.Redirect("/signin?redirect=" + url.QueryEscape(c.Request.URL.RequestURI()))
				}
				return c.Unauthorized()
			}
			return next(c)
		}
	}
}

// IsAuthorized blocks non-authorized requests
func IsAuthorized(roles ...enum.Role) web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			user := c.User()
			for _, role := range roles {
				if user.Role == role {
					return next(c)
				}
			}
			return c.Forbidden()
		}
	}
}
