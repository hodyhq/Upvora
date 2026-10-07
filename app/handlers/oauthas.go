package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/oauthas"
	"github.com/getfider/fider/app/pkg/ratelimit"
	"github.com/getfider/fider/app/pkg/web"
)

// Embedded OAuth 2.1 authorization server for MCP clients. Every endpoint is
// off (404) unless the admin enabled MCP for the site.

var (
	oauthRegisterPerIP     = ratelimit.New(20, time.Hour)
	oauthRegisterPerTenant = ratelimit.New(200, time.Hour)
	oauthTokenPerIP        = ratelimit.New(120, time.Minute)
)

// ResetOAuthRateLimits clears the limiters (tests only).
func ResetOAuthRateLimits() {
	oauthRegisterPerIP = ratelimit.New(20, time.Hour)
	oauthRegisterPerTenant = ratelimit.New(200, time.Hour)
	oauthTokenPerIP = ratelimit.New(120, time.Minute)
}

func mcpEnabled(c *web.Context) bool {
	return c.Tenant() != nil && c.Tenant().MCPEnabled
}

// MCPResourceURL is the protected resource (and access-token audience) for
// this site.
func MCPResourceURL(c *web.Context) string {
	return c.BaseURL() + "/mcp"
}

func oauthError(c *web.Context, status int, code, description string) error {
	return c.JSON(status, web.Map{"error": code, "error_description": description})
}

// OAuthAuthorizationServerMetadata serves RFC 8414 discovery.
func OAuthAuthorizationServerMetadata() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mcpEnabled(c) {
			return c.NotFound()
		}
		base := c.BaseURL()
		meta := web.Map{
			"issuer":                                base,
			"authorization_endpoint":                base + "/oauth2/authorize",
			"token_endpoint":                        base + "/oauth2/token",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
			"scopes_supported":                      []string{oauthas.ScopeFull, oauthas.ScopeRead},
		}
		if c.Tenant().MCPDCREnabled {
			meta["registration_endpoint"] = base + "/oauth2/register"
		}
		return c.Ok(meta)
	}
}

// OAuthProtectedResourceMetadata serves RFC 9728 discovery for /mcp.
func OAuthProtectedResourceMetadata() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mcpEnabled(c) {
			return c.NotFound()
		}
		return c.Ok(web.Map{
			"resource":                 MCPResourceURL(c),
			"authorization_servers":    []string{c.BaseURL()},
			"scopes_supported":         []string{oauthas.ScopeFull, oauthas.ScopeRead},
			"bearer_methods_supported": []string{"header"},
		})
	}
}

// OAuthRegister is RFC 7591 dynamic client registration for public clients.
// Registering grants nothing: the user still signs in and consents.
func OAuthRegister() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mcpEnabled(c) {
			return c.NotFound()
		}
		if !c.Tenant().MCPDCREnabled {
			return oauthError(c, http.StatusForbidden, "access_denied", "Client registration is disabled on this site; ask an administrator to register the client.")
		}
		tenantKey := strconv.Itoa(c.Tenant().ID)
		if !oauthRegisterPerIP.Allow(tenantKey+"|"+c.Request.ClientIP()) || !oauthRegisterPerTenant.Allow(tenantKey) {
			return oauthError(c, http.StatusTooManyRequests, "slow_down", "Too many registrations; try again later.")
		}

		var input struct {
			ClientName              string   `json:"client_name"`
			RedirectURIs            []string `json:"redirect_uris"`
			TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		}
		if err := json.Unmarshal([]byte(c.Request.Body), &input); err != nil {
			return oauthError(c, http.StatusBadRequest, "invalid_client_metadata", "Body must be a JSON object.")
		}
		if input.TokenEndpointAuthMethod != "" && input.TokenEndpointAuthMethod != "none" {
			return oauthError(c, http.StatusBadRequest, "invalid_client_metadata", "Only public clients (token_endpoint_auth_method none) are supported.")
		}
		if len(input.RedirectURIs) == 0 || len(input.RedirectURIs) > 10 {
			return oauthError(c, http.StatusBadRequest, "invalid_redirect_uri", "Provide 1 to 10 redirect_uris.")
		}
		for _, uri := range input.RedirectURIs {
			if len(uri) > 2000 || !oauthas.ValidRedirectURI(uri) {
				return oauthError(c, http.StatusBadRequest, "invalid_redirect_uri", "Redirect URIs must be https, http on a loopback host, or a native app scheme, without a fragment.")
			}
		}
		name := strings.TrimSpace(input.ClientName)
		if name == "" {
			name = "MCP client"
		}
		if len(name) > 100 {
			name = name[:100]
		}

		reg := &cmd.RegisterOAuthClient{Name: name, RedirectURIs: input.RedirectURIs}
		if err := bus.Dispatch(c, reg); err != nil {
			return c.Failure(err)
		}
		return c.JSON(http.StatusCreated, web.Map{
			"client_id":                  reg.Result.ClientID,
			"client_name":                reg.Result.Name,
			"redirect_uris":              reg.Result.RedirectURIs,
			"token_endpoint_auth_method": "none",
			"grant_types":                []string{"authorization_code", "refresh_token"},
			"response_types":             []string{"code"},
			"client_id_issued_at":        time.Now().Unix(),
		})
	}
}
