package middlewares

import (
	"crypto/subtle"
	"fmt"
	"github.com/getfider/fider/app/pkg/oauthas"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"

	"github.com/getfider/fider/app/pkg/validate"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
	webutil "github.com/getfider/fider/app/pkg/web/util"
)

// User gets JWT Auth token from cookie and insert into context
func User() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			var (
				token            string
				user             *entity.User
				fromSignUpCookie bool
				fromAPIKey       bool
			)

			// An MCP access token on /api or /mcp is the caller's identity; a
			// browser cookie sent alongside it is ignored, so a stale cookie
			// can never shadow or replace the token.
			bearer, hasBearer := web.ParseBearerToken(c.Request.GetHeader("Authorization"))
			mcpBearer := hasBearer && isMCPAccessToken(bearer) && (c.Request.IsAPI() || isMCPPath(c))

			// /mcp takes only Bearer tokens: a browser session there would list
			// tools that then run anonymously.
			cookieAllowed := !mcpBearer && !isMCPPath(c)
			cookie, err := c.Request.Cookie(web.CookieAuthName)
			if !cookieAllowed {
				err = http.ErrNoCookie
			}
			if err == nil {
				token = cookie.Value
			} else if cookieAllowed {
				// The signup-transfer cookie is domain-wide, so it reaches every tenant
				// subdomain. We do NOT promote it to a durable host-only auth cookie here:
				// that only happens later, and only once we have confirmed the token's user
				// actually belongs to the tenant selected by the current host.
				token = webutil.GetSignUpAuthCookie(c)
				fromSignUpCookie = token != ""
			}

			if token != "" {
				claims, err := jwt.DecodeFiderClaims(token)
				if err != nil {
					c.RemoveCookie(web.CookieAuthName)
					return next(c)
				}

				// Scope the lookup to the tenant selected by the request host. A globally
				// valid user ID that belongs to a different tenant must resolve to "not
				// found" so a token cannot be replayed across tenant boundaries.
				tenantID := 0
				if c.Tenant() != nil {
					tenantID = c.Tenant().ID
				}
				userByClaimsID := &query.GetUserByID{UserID: claims.UserID, TenantID: tenantID}
				err = bus.Dispatch(c, userByClaimsID)
				user = userByClaimsID.Result
				if err != nil {
					if errors.Cause(err) == app.ErrNotFound {
						c.RemoveCookie(web.CookieAuthName)
						return next(c)
					}
					return err
				}

				// Security stamp check: if the JWT contains a stamp (new tokens only),
				// validate it against the current DB stamp. A mismatch means the user's
				// security-relevant data has changed (e.g. role changed, account blocked,
				// or OAuth allowed-roles updated) and they must re-authenticate so that
				// access controls are re-evaluated.
				if claims.SecurityStamp != "" && user != nil && claims.SecurityStamp != user.SecurityStamp {
					c.RemoveCookie(web.CookieAuthName)
					if c.IsAjax() {
						return c.JSON(401, web.Map{})
					}
					redirectTarget := c.Request.URL.RequestURI()
					if redirectTarget != "" &&
						redirectTarget != "/" &&
						!strings.HasPrefix(redirectTarget, "/signin") &&
						!strings.HasPrefix(redirectTarget, "/signout") {
						return c.Redirect("/signin?redirect=" + url.QueryEscape(redirectTarget))
					}
					return c.Redirect("/signin")
				}
			} else if mcpBearer {
				// OAuth access token issued to an MCP client by this site.
				mcpUser, scope, status := userFromMCPAccessToken(c, bearer)
				if status != 0 {
					return mcpUnauthorized(c, status)
				}
				user = mcpUser
				c.Set(oauthas.ScopeCtxKey{}, scope)
			} else if c.Request.IsAPI() {
				if apiKey, ok := web.ParseBearerToken(c.Request.GetHeader("Authorization")); ok {
					getUserByAPIKey := &query.GetUserByAPIKey{APIKey: apiKey}
					err = bus.Dispatch(c, getUserByAPIKey)
					if err != nil {
						if errors.Cause(err) == app.ErrNotFound {
							return c.HandleValidation(validate.Failed("API Key is invalid"))
						}
						return err
					}
					user = getUserByAPIKey.Result
					fromAPIKey = true

					if !user.IsCollaborator() {
						return c.HandleValidation(validate.Failed("API Key is invalid"))
					}

					if impersonateUserIDStr := c.Request.GetHeader("X-Fider-UserID"); impersonateUserIDStr != "" {
						if !user.IsAdministrator() {
							return c.HandleValidation(validate.Failed("Only Administrators are allowed to impersonate another user"))
						}
						impersonateUserID, err := strconv.Atoi(impersonateUserIDStr)
						if err != nil {
							return c.HandleValidation(validate.Failed(fmt.Sprintf("User not found for given impersonate UserID '%s'", impersonateUserIDStr)))
						}
						userByImpersonateID := &query.GetUserByID{UserID: impersonateUserID, TenantID: user.Tenant.ID}
						err = bus.Dispatch(c, userByImpersonateID)
						user = userByImpersonateID.Result
						if err != nil {
							if errors.Cause(err) == app.ErrNotFound {
								return c.HandleValidation(validate.Failed(fmt.Sprintf("User not found for given impersonate UserID '%s'", impersonateUserIDStr)))
							}
							return err
						}
						// Never act as another administrator: owner-only actions (site
						// deletion) and key rotation trust c.User(), so impersonating an
						// admin would let one admin act as the owner or mint their key.
						if user.IsAdministrator() {
							return c.HandleValidation(validate.Failed("Administrators cannot be impersonated"))
						}
					}
				}
			}

			if user != nil && c.Tenant() != nil && user.Tenant.ID == c.Tenant().ID {
				// blocked users are unable to sign in
				if user.Status == enum.UserBlocked {
					c.RemoveCookie(web.CookieAuthName)
					return c.Unauthorized()
				}

				// Only now that the user is confirmed to belong to the current tenant do we
				// promote a signup-transfer cookie into a durable host-only auth cookie.
				// This is deliberately skipped on any other tenant, so the domain-wide
				// signup token cannot become a normal session on an unrelated subdomain.
				if fromSignUpCookie {
					webutil.AddAuthTokenCookie(c, token)
				}

				c.SetUser(user)
				if fromAPIKey {
					c.SetAuthenticatedByAPIKey()
				}
			}

			if user == nil && isMCPPath(c) {
				return mcpUnauthorized(c, http.StatusUnauthorized)
			}

			return next(c)
		}
	}
}

// isMCPAccessToken reports whether a Bearer value is a JWT (API keys never
// contain dots), so it is checked as an access token and never as an API key.
func isMCPAccessToken(bearer string) bool {
	return strings.Count(bearer, ".") == 2
}

func isMCPPath(c *web.Context) bool {
	path := c.Request.URL.Path
	return path == "/mcp" || strings.HasPrefix(path, "/mcp/")
}

// userFromMCPAccessToken resolves the user of an MCP access token, applying
// the site's current MCP rules on every request: the token must be for this
// site and tenant, the security stamp must still match (role change, block or
// sign-out revoke it), MCP must be on, and the role must meet the minimum.
// A read-only token may only make safe requests. Returns a status on refusal.
func userFromMCPAccessToken(c *web.Context, token string) (*entity.User, string, int) {
	tenant := c.Tenant()
	if tenant == nil || !tenant.MCPEnabled {
		return nil, "", http.StatusUnauthorized
	}
	claims, err := jwt.DecodeMCPAccessClaims(token, c.BaseURL()+"/mcp")
	if err != nil || claims.TenantID != tenant.ID {
		return nil, "", http.StatusUnauthorized
	}
	getUser := &query.GetUserByID{UserID: claims.UserID, TenantID: tenant.ID}
	if bus.Dispatch(c, getUser) != nil {
		return nil, "", http.StatusUnauthorized
	}
	user := getUser.Result
	if subtle.ConstantTimeCompare([]byte(user.SecurityStamp), []byte(claims.SecurityStamp)) != 1 ||
		user.Status != enum.UserActive || user.Role < tenant.MCPMinRole {
		return nil, "", http.StatusUnauthorized
	}
	// /mcp itself is all POSTs; there, write tools are hidden and each replayed
	// call is checked here again with its real method.
	if claims.Scope == oauthas.ScopeRead && !isMCPPath(c) && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return nil, "", http.StatusForbidden
	}
	return user, claims.Scope, 0
}

// mcpUnauthorized refuses an MCP or token request; a 401 carries the RFC 9728
// pointer clients use to discover how to authorize.
func mcpUnauthorized(c *web.Context, status int) error {
	if status == http.StatusUnauthorized {
		c.Response.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+c.BaseURL()+`/.well-known/oauth-protected-resource"`)
	}
	return c.JSON(status, web.Map{})
}
