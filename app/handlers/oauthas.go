package handlers

import (
	"encoding/json"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/rand"
	"net/http"
	"net/url"
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

// authorizeParams are the authorization request fields, from the query string
// (GET) or the consent page's JSON (POST). The POST is validated again in full.
type authorizeParams struct {
	ClientID            string `json:"clientId"`
	RedirectURI         string `json:"redirectUri"`
	ResponseType        string `json:"responseType"`
	CodeChallenge       string `json:"codeChallenge"`
	CodeChallengeMethod string `json:"codeChallengeMethod"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	Approve             bool   `json:"approve"`
}

// authorizeCheck is the outcome of validating a request: either a page error
// (client or redirect not verified, so we must not redirect), a protocol error
// for the verified redirect, a role denial, or a valid request.
type authorizeCheck struct {
	client     *entity.OAuthClient
	scope      string
	pageError  string // shown on our page, never redirected
	oauthError string // sent back to the verified redirect_uri
	denied     bool   // signed-in user's role is below the site's minimum
}

func checkAuthorize(c *web.Context, p authorizeParams) authorizeCheck {
	getClient := &query.GetOAuthClient{ClientID: p.ClientID}
	if p.ClientID == "" || bus.Dispatch(c, getClient) != nil {
		return authorizeCheck{pageError: "This application is not registered on this site."}
	}
	matched := false
	for _, uri := range getClient.Result.RedirectURIs {
		if uri == p.RedirectURI {
			matched = true
		}
	}
	if !matched {
		return authorizeCheck{pageError: "The application's return address does not match its registration."}
	}
	check := authorizeCheck{client: getClient.Result}
	if p.ResponseType != "code" {
		check.oauthError = "unsupported_response_type"
		return check
	}
	if p.CodeChallengeMethod != "S256" || !oauthas.ValidChallenge(p.CodeChallenge) {
		check.oauthError = "invalid_request"
		return check
	}
	scope, ok := oauthas.ParseScope(p.Scope)
	if !ok {
		check.oauthError = "invalid_scope"
		return check
	}
	check.scope = scope
	if c.User().Role < c.Tenant().MCPMinRole {
		check.denied = true
	}
	return check
}

// clientRedirect appends params to a verified redirect URI, keeping its own query.
func clientRedirect(redirectURI string, params map[string]string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

const roleDeniedMessage = "Your role on this site is not allowed to connect MCP clients."

func consentPage(c *web.Context, status int, data web.Map) error {
	return c.Page(status, web.Props{
		Page:  "OAuthConsent/OAuthConsent.page",
		Title: "Connect an application",
		Data:  data,
	})
}

// OAuthAuthorize validates an authorization request and shows the consent
// screen. Signing in is handled before this by IsAuthenticated (the instance's
// own sign-in, whatever providers it has).
func OAuthAuthorize() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mcpEnabled(c) {
			return c.NotFound()
		}
		p := authorizeParams{
			ClientID:            c.QueryParam("client_id"),
			RedirectURI:         c.QueryParam("redirect_uri"),
			ResponseType:        c.QueryParam("response_type"),
			CodeChallenge:       c.QueryParam("code_challenge"),
			CodeChallengeMethod: c.QueryParam("code_challenge_method"),
			Scope:               c.QueryParam("scope"),
			State:               c.QueryParam("state"),
		}
		check := checkAuthorize(c, p)
		switch {
		case check.pageError != "":
			return consentPage(c, http.StatusBadRequest, web.Map{"error": check.pageError})
		case check.oauthError != "":
			return c.Redirect(clientRedirect(p.RedirectURI, map[string]string{"error": check.oauthError, "state": p.State, "iss": c.BaseURL()}))
		case check.denied:
			return consentPage(c, http.StatusForbidden, web.Map{"error": roleDeniedMessage})
		}
		redirectHost := ""
		if u, err := url.Parse(p.RedirectURI); err == nil {
			redirectHost = u.Host
			if redirectHost == "" {
				redirectHost = u.Scheme
			}
		}
		return consentPage(c, http.StatusOK, web.Map{
			"clientName":          check.client.Name,
			"redirectHost":        redirectHost,
			"scope":               check.scope,
			"clientId":            p.ClientID,
			"redirectUri":         p.RedirectURI,
			"codeChallenge":       p.CodeChallenge,
			"codeChallengeMethod": p.CodeChallengeMethod,
			"state":               p.State,
		})
	}
}

// OAuthAuthorizeDecision records the user's approve/deny choice (same-origin
// JSON, CSRF-protected) and returns where the browser should go next.
func OAuthAuthorizeDecision() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mcpEnabled(c) {
			return c.NotFound()
		}
		p := authorizeParams{}
		if err := json.Unmarshal([]byte(c.Request.Body), &p); err != nil {
			return c.BadRequest(web.Map{"error": "invalid request"})
		}
		p.ResponseType = "code"
		check := checkAuthorize(c, p)
		switch {
		case check.pageError != "" || check.oauthError != "":
			return c.BadRequest(web.Map{"error": "invalid authorization request"})
		case check.denied:
			return c.JSON(http.StatusForbidden, web.Map{"error": roleDeniedMessage})
		}
		if !p.Approve {
			return c.Ok(web.Map{"redirect": clientRedirect(p.RedirectURI, map[string]string{"error": "access_denied", "state": p.State, "iss": c.BaseURL()})})
		}

		plain, hash := oauthas.NewToken()
		if err := bus.Dispatch(c, &cmd.SaveOAuthCode{
			CodeHash:      hash,
			ClientID:      check.client.ClientID,
			UserID:        c.User().ID,
			RedirectURI:   p.RedirectURI,
			Scope:         check.scope,
			CodeChallenge: p.CodeChallenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		}); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"redirect": clientRedirect(p.RedirectURI, map[string]string{"code": plain, "state": p.State, "iss": c.BaseURL()})})
	}
}

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

// OAuthTokenEndpoint exchanges an authorization code (with its PKCE verifier)
// or a refresh token for an access token. Every failure is an RFC 6749 error
// response, never a Go error: the request transaction must commit so that a
// burned code or a revoked refresh family stays revoked.
func OAuthTokenEndpoint() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mcpEnabled(c) {
			return c.NotFound()
		}
		c.Response.Header().Set("Cache-Control", "no-store")
		c.Response.Header().Set("Pragma", "no-cache")
		if !oauthTokenPerIP.Allow(strconv.Itoa(c.Tenant().ID) + "|" + c.Request.ClientIP()) {
			return oauthError(c, http.StatusTooManyRequests, "slow_down", "Too many token requests; try again later.")
		}
		form, err := url.ParseQuery(c.Request.Body)
		if err != nil {
			return oauthError(c, http.StatusBadRequest, "invalid_request", "Body must be form-encoded.")
		}
		getClient := &query.GetOAuthClient{ClientID: form.Get("client_id")}
		if form.Get("client_id") == "" || bus.Dispatch(c, getClient) != nil {
			return oauthError(c, http.StatusUnauthorized, "invalid_client", "Unknown client.")
		}

		var grant *entity.OAuthGrant
		var familyID string
		newRefresh, newRefreshHash := oauthas.NewToken()
		switch form.Get("grant_type") {
		case "authorization_code":
			consume := &cmd.ConsumeOAuthCode{CodeHash: oauthas.HashToken(form.Get("code")), ClientID: getClient.Result.ClientID}
			if bus.Dispatch(c, consume) != nil {
				return oauthError(c, http.StatusBadRequest, "invalid_grant", "The code is invalid, expired, or already used.")
			}
			grant = consume.Result
			// The code is burned whatever happens next.
			if form.Get("redirect_uri") != grant.RedirectURI ||
				!oauthas.VerifyPKCE(form.Get("code_verifier"), grant.CodeChallenge, "S256") {
				return oauthError(c, http.StatusBadRequest, "invalid_grant", "The redirect URI or code verifier does not match.")
			}
			familyID = rand.String(32)
		case "refresh_token":
			rotate := &cmd.RotateOAuthRefreshToken{
				OldHash:      oauthas.HashToken(form.Get("refresh_token")),
				NewHash:      newRefreshHash,
				ClientID:     getClient.Result.ClientID,
				NewExpiresAt: time.Now().Add(refreshTokenTTL),
			}
			if bus.Dispatch(c, rotate) != nil {
				return oauthError(c, http.StatusBadRequest, "invalid_grant", "The refresh token is invalid, expired, or already used.")
			}
			grant = rotate.Result
		default:
			return oauthError(c, http.StatusBadRequest, "unsupported_grant_type", "Use authorization_code or refresh_token.")
		}

		// Re-check the user against the site's current rules.
		getUser := &query.GetUserByID{UserID: grant.UserID, TenantID: c.Tenant().ID}
		if bus.Dispatch(c, getUser) != nil || getUser.Result.Status != enum.UserActive || getUser.Result.Role < c.Tenant().MCPMinRole {
			return oauthError(c, http.StatusBadRequest, "invalid_grant", "This account cannot use MCP on this site.")
		}
		user := getUser.Result

		if familyID != "" {
			if err := bus.Dispatch(c, &cmd.SaveOAuthRefreshToken{
				TokenHash: newRefreshHash,
				ClientID:  grant.ClientID,
				UserID:    user.ID,
				Scope:     grant.Scope,
				FamilyID:  familyID,
				ExpiresAt: time.Now().Add(refreshTokenTTL),
			}); err != nil {
				return c.Failure(err)
			}
		}

		now := time.Now()
		access, err := jwt.Encode(&jwt.MCPAccessClaims{
			UserID:        user.ID,
			TenantID:      c.Tenant().ID,
			ClientID:      grant.ClientID,
			Scope:         grant.Scope,
			SecurityStamp: user.SecurityStamp,
			Metadata: jwt.Metadata{
				Issuer:    c.BaseURL(),
				Subject:   strconv.Itoa(user.ID),
				Audience:  []string{MCPResourceURL(c)},
				IssuedAt:  jwt.Time(now),
				ExpiresAt: jwt.Time(now.Add(accessTokenTTL)),
			},
		})
		if err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{
			"access_token":  access,
			"token_type":    "Bearer",
			"expires_in":    int(accessTokenTTL.Seconds()),
			"refresh_token": newRefresh,
			"scope":         grant.Scope,
		})
	}
}
