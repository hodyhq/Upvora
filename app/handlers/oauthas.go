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

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
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
	oauthTokenPerTenant    = ratelimit.New(1000, time.Minute)
)

// ResetOAuthRateLimits clears the limiters (tests only).
func ResetOAuthRateLimits() {
	oauthRegisterPerIP = ratelimit.New(20, time.Hour)
	oauthRegisterPerTenant = ratelimit.New(200, time.Hour)
	oauthTokenPerIP = ratelimit.New(120, time.Minute)
	oauthTokenPerTenant = ratelimit.New(1000, time.Minute)
}

func mcpEnabled(c *web.Context) bool {
	return c.Tenant() != nil && c.Tenant().MCPEnabled
}

// MCPConsentCookie holds the consent-step identity on an MCP-only address.
const MCPConsentCookie = web.CookieMCPConsentName

const (
	mcpSignInRequestTTL = 10 * time.Minute // signing in on the board
	mcpHandoffTTL       = 2 * time.Minute  // the trip back to the MCP-only address
	mcpConsentTTL       = 15 * time.Minute // the consent step there
)

// mcpOrigin returns the MCP-only public address (MCP_ORIGINS) this request
// arrived on, in canonical form.
func mcpOrigin(c *web.Context) (string, bool) {
	if !env.IsSingleHostMode() {
		return "", false
	}
	return env.MatchMCPOrigin(c.BaseURL())
}

// publicMCPURLs are the MCP endpoints on the MCP-only public addresses.
func publicMCPURLs() []string {
	urls := []string{}
	if env.IsSingleHostMode() {
		for _, origin := range env.Config.MCPOrigins {
			urls = append(urls, origin+"/mcp")
		}
	}
	return urls
}

// grantOrigin is the address codes and refresh tokens are bound to: the
// MCP-only address in canonical form, or this request's own address.
func grantOrigin(c *web.Context) string {
	if origin, ok := mcpOrigin(c); ok {
		return origin
	}
	return c.BaseURL()
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
		if runes := []rune(name); len(runes) > 100 {
			name = string(runes[:100])
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
		if !c.IsAuthenticated() {
			// On an MCP-only address, sign in on the board's own address (which
			// stays internal) and come back: see OAuthSignInContinue.
			if origin, ok := mcpOrigin(c); ok {
				token, err := jwt.EncodeMCPSignInRequest(c.Tenant().ID, origin, c.Request.URL.RawQuery, mcpSignInRequestTTL)
				if err != nil {
					return c.Failure(err)
				}
				return c.Redirect(web.BaseURL(c) + "/oauth2/continue?request=" + url.QueryEscape(token))
			}
			return c.Redirect("/signin?redirect=" + url.QueryEscape(c.Request.URL.RequestURI()))
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
		connectingThrough := any(nil)
		if _, ok := mcpOrigin(c); ok {
			connectingThrough = c.Request.URL.Host
		}
		return consentPage(c, http.StatusOK, web.Map{
			"connectingThrough":   connectingThrough,
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
		// The consent-step identity on an MCP-only address is spent either way.
		if _, ok := mcpOrigin(c); ok {
			c.RemoveCookie(MCPConsentCookie)
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
			SecurityStamp: c.User().SecurityStamp,
			Origin:        grantOrigin(c),
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
		tenantKey := strconv.Itoa(c.Tenant().ID)
		if !oauthTokenPerIP.Allow(tenantKey+"|"+c.Request.ClientIP()) || !oauthTokenPerTenant.Allow(tenantKey) {
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
		// Codes and refresh tokens work only on the address they were issued on.
		// Ones issued before they carried an address work on the board's own.
		origin := grantOrigin(c)
		_, onMCP := mcpOrigin(c)
		switch form.Get("grant_type") {
		case "authorization_code":
			consume := &cmd.ConsumeOAuthCode{CodeHash: oauthas.HashToken(form.Get("code")), ClientID: getClient.Result.ClientID, Origin: origin, AllowUnbound: !onMCP}
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
				Origin:       origin,
				AllowUnbound: !onMCP,
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
				TokenHash:     newRefreshHash,
				ClientID:      grant.ClientID,
				UserID:        user.ID,
				Scope:         grant.Scope,
				FamilyID:      familyID,
				SecurityStamp: user.SecurityStamp,
				FromCodeHash:  oauthas.HashToken(form.Get("code")),
				Origin:        origin,
				ExpiresAt:     time.Now().Add(refreshTokenTTL),
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

// ManageMCPPage renders the admin MCP page (settings ride on the tenant).
func ManageMCPPage() web.HandlerFunc {
	return func(c *web.Context) error {
		clients := &query.ListOAuthClients{}
		if err := bus.Dispatch(c, clients); err != nil {
			return c.Failure(err)
		}
		return c.Page(http.StatusOK, web.Props{
			Page:  "Administration/pages/ManageMCP.page",
			Title: "MCP · Site Settings",
			Data:  web.Map{"clients": clients.Result, "mcpUrl": MCPResourceURL(c), "publicMcpUrls": publicMCPURLs()},
		})
	}
}

// ListMCPClients lists registered MCP clients (admin).
func ListMCPClients() web.HandlerFunc {
	return func(c *web.Context) error {
		if !c.User().IsAdministrator() {
			return c.Forbidden()
		}
		clients := &query.ListOAuthClients{}
		if err := bus.Dispatch(c, clients); err != nil {
			return c.Failure(err)
		}
		return c.Ok(clients.Result)
	}
}

// CreateMCPClient pre-registers a client (admin), e.g. when self-registration is off.
func CreateMCPClient() web.HandlerFunc {
	return func(c *web.Context) error {
		if !c.User().IsAdministrator() {
			return c.Forbidden()
		}
		var input struct {
			Name         string   `json:"name"`
			RedirectURIs []string `json:"redirectUris"`
		}
		if err := json.Unmarshal([]byte(c.Request.Body), &input); err != nil {
			return c.BadRequest(web.Map{"error": "invalid request"})
		}
		name := strings.TrimSpace(input.Name)
		if name == "" || len(name) > 100 {
			return c.BadRequest(web.Map{"name": "Name is required (up to 100 characters)."})
		}
		if len(input.RedirectURIs) == 0 || len(input.RedirectURIs) > 10 {
			return c.BadRequest(web.Map{"redirectUris": "Provide 1 to 10 redirect URIs."})
		}
		for _, uri := range input.RedirectURIs {
			if len(uri) > 2000 || !oauthas.ValidRedirectURI(uri) {
				return c.BadRequest(web.Map{"redirectUris": "Redirect URIs must be https, http on a loopback host, or a native app scheme, without a fragment."})
			}
		}
		reg := &cmd.RegisterOAuthClient{Name: name, RedirectURIs: input.RedirectURIs, CreatedByAdmin: true}
		if err := bus.Dispatch(c, reg); err != nil {
			return c.Failure(err)
		}
		return c.Ok(reg.Result)
	}
}

// DeleteMCPClient removes a client and revokes everything issued to it (admin).
func DeleteMCPClient() web.HandlerFunc {
	return func(c *web.Context) error {
		if !c.User().IsAdministrator() {
			return c.Forbidden()
		}
		id, err := c.ParamAsInt("id")
		if err != nil {
			return c.NotFound()
		}
		if err := bus.Dispatch(c, &cmd.DeleteOAuthClient{ID: id}); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}

const expiredConnectMessage = "This connection request is invalid or has expired. Start connecting again from your assistant."

// OAuthSignInContinue runs on the board's own address, after its normal
// sign-in, for a connection started on an MCP-only address. It verifies the
// signed request, re-checks that its address is configured (never redirecting
// anywhere else), and sends the browser back there with a single-use code.
func OAuthSignInContinue() web.HandlerFunc {
	return func(c *web.Context) error {
		if _, onMCP := mcpOrigin(c); !mcpEnabled(c) || onMCP {
			return c.NotFound()
		}
		claims, err := jwt.DecodeMCPSignInRequest(c.QueryParam("request"))
		if err != nil || claims.TenantID != c.Tenant().ID {
			return consentPage(c, http.StatusBadRequest, web.Map{"error": expiredConnectMessage})
		}
		origin, ok := env.MatchMCPOrigin(claims.Origin)
		if !ok {
			return consentPage(c, http.StatusBadRequest, web.Map{"error": expiredConnectMessage})
		}
		plain, hash := oauthas.NewToken()
		if err := bus.Dispatch(c, &cmd.SaveOAuthSignInHandoff{
			CodeHash:      hash,
			UserID:        c.User().ID,
			Origin:        origin,
			Query:         claims.Query,
			SecurityStamp: c.User().SecurityStamp,
			ExpiresAt:     time.Now().Add(mcpHandoffTTL),
		}); err != nil {
			return c.Failure(err)
		}
		return c.Redirect(origin + "/oauth2/resume?code=" + url.QueryEscape(plain))
	}
}

// OAuthResume runs on an MCP-only address: it redeems the single-use code from
// OAuthSignInContinue and sets a short-lived cookie that identifies the person
// for the consent step only, then resumes the authorization request.
func OAuthResume() web.HandlerFunc {
	return func(c *web.Context) error {
		origin, ok := mcpOrigin(c)
		if !mcpEnabled(c) || !ok {
			return c.NotFound()
		}
		redeem := &cmd.RedeemOAuthSignInHandoff{CodeHash: oauthas.HashToken(c.QueryParam("code")), Origin: origin}
		if err := bus.Dispatch(c, redeem); err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return consentPage(c, http.StatusBadRequest, web.Map{"error": expiredConnectMessage})
			}
			return c.Failure(err)
		}
		getUser := &query.GetUserByID{UserID: redeem.Result.UserID, TenantID: c.Tenant().ID}
		if bus.Dispatch(c, getUser) != nil || getUser.Result.Status != enum.UserActive {
			return consentPage(c, http.StatusBadRequest, web.Map{"error": expiredConnectMessage})
		}
		user := getUser.Result
		token, err := jwt.EncodeMCPConsent(user.ID, c.Tenant().ID, user.SecurityStamp, origin, mcpConsentTTL)
		if err != nil {
			return c.Failure(err)
		}
		c.AddCookie(MCPConsentCookie, token, time.Now().Add(mcpConsentTTL))
		return c.Redirect("/oauth2/authorize?" + redeem.Result.Query)
	}
}
