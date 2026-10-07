package handlers_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/oauthas"
)

const (
	boardBase = "https://board.demo.test"
	mcpBase   = "https://mcp.demo.test"
)

// withMCPOrigin runs a test as a single-site board at boardBase that also
// serves the MCP connection on mcpBase.
func withMCPOrigin(t *testing.T) {
	mode, base := env.Config.HostMode, env.Config.BaseURL
	env.Config.HostMode, env.Config.BaseURL = "single", boardBase
	env.Config.MCPOrigins = []string{mcpBase}
	t.Cleanup(func() {
		env.Config.HostMode, env.Config.BaseURL, env.Config.MCPOrigins = mode, base, nil
	})
}

func authorizeOn(base string) string {
	return strings.Replace(authorizeQuery(nil), "http://demo.test.fider.io", base, 1)
}

func TestOAuthAuthorize_SignedOutOnBoardGoesToSignIn(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()
	target := authorizeQuery(nil)
	code, res := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).WithURL(target).Execute(handlers.OAuthAuthorize())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	u, _ := url.Parse(target)
	Expect(res.Header().Get("Location")).Equals("/signin?redirect=" + url.QueryEscape(u.RequestURI()))
}

func TestOAuthAuthorize_SignedOutOnMCPOriginSignsInOnTheBoard(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	mockOAuthClient()
	target := authorizeOn(mcpBase)
	code, res := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).WithURL(target).Execute(handlers.OAuthAuthorize())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	loc, _ := url.Parse(res.Header().Get("Location"))
	Expect(loc.Scheme + "://" + loc.Host + loc.Path).Equals(boardBase + "/oauth2/continue")
	claims, err := jwt.DecodeMCPSignInRequest(loc.Query().Get("request"))
	Expect(err).IsNil()
	Expect(claims.Origin).Equals(mcpBase)
	// the browser that starts the sign-in gets a flow cookie; only it can finish
	var flow *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == handlers.MCPFlowCookie {
			flow = c
		}
	}
	Expect(flow).IsNotNil()
	Expect(flow.HttpOnly).IsTrue()
	Expect(claims.FlowHash).Equals(oauthas.HashToken(flow.Value))
	want, _ := url.Parse(target)
	Expect(claims.Query).Equals(want.RawQuery)
}

func signInRequest(origin string, tenantID int) string {
	token, _ := jwt.EncodeMCPSignInRequest(tenantID, origin, "client_id=c&state=s", "flow-hash", time.Minute)
	return boardBase + "/oauth2/continue?request=" + url.QueryEscape(token)
}

func TestOAuthSignInContinue_ReturnsToTheMCPOriginWithAHandoff(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	saved := []*cmd.SaveOAuthSignInHandoff{}
	bus.AddHandler(func(ctx context.Context, c *cmd.SaveOAuthSignInHandoff) error {
		saved = append(saved, c)
		return nil
	})
	tenant := mcpTenantMinRole(enum.RoleVisitor)
	code, res := mock.NewSingleTenantServer().OnTenant(tenant).AsUser(mock.AryaStark).
		WithURL(signInRequest(mcpBase, tenant.ID)).Execute(handlers.OAuthSignInContinue())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	loc, _ := url.Parse(res.Header().Get("Location"))
	Expect(loc.Scheme + "://" + loc.Host + loc.Path).Equals(mcpBase + "/oauth2/resume")
	Expect(saved).HasLen(1)
	Expect(saved[0].UserID).Equals(mock.AryaStark.ID)
	Expect(saved[0].Origin).Equals(mcpBase)
	Expect(saved[0].Query).Equals("client_id=c&state=s")
	Expect(saved[0].FlowHash).Equals("flow-hash")
	Expect(saved[0].CodeHash == loc.Query().Get("code")).IsFalse() // only the hash is stored
	Expect(len(loc.Query().Get("code")) >= 43).IsTrue()
}

func TestOAuthSignInContinue_Refusals(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	bus.AddHandler(func(ctx context.Context, c *cmd.SaveOAuthSignInHandoff) error { return nil })
	tenant := mcpTenantMinRole(enum.RoleVisitor)
	for _, target := range []string{
		signInRequest("https://evil.example.com", tenant.ID), // not a configured address: never redirected to
		signInRequest(mcpBase, tenant.ID+1),                  // another site
		boardBase + "/oauth2/continue?request=garbage",
	} {
		code, res := mock.NewSingleTenantServer().OnTenant(tenant).AsUser(mock.AryaStark).WithURL(target).Execute(handlers.OAuthSignInContinue())
		Expect(code).Equals(http.StatusBadRequest)
		Expect(res.Header().Get("Location")).Equals("")
	}
	code, _ := mock.NewSingleTenantServer().OnTenant(mcpTenant(false, true)).AsUser(mock.AryaStark).
		WithURL(signInRequest(mcpBase, tenant.ID)).Execute(handlers.OAuthSignInContinue())
	Expect(code).Equals(http.StatusNotFound)
}

func TestOAuthResume_SetsAConsentOnlyCookie(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	var redeemed *cmd.RedeemOAuthSignInHandoff
	bus.AddHandler(func(ctx context.Context, c *cmd.RedeemOAuthSignInHandoff) error {
		redeemed = c
		c.Result = &entity.OAuthSignInHandoff{UserID: mock.AryaStark.ID, Query: "client_id=c&state=s"}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		q.Result = mock.AryaStark
		return nil
	})
	code, res := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).
		AddCookie(handlers.MCPFlowCookie, "plain-flow").
		WithURL(mcpBase + "/oauth2/resume?code=abc").Execute(handlers.OAuthResume())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(res.Header().Get("Location")).Equals("/oauth2/authorize?client_id=c&state=s")
	Expect(redeemed.Origin).Equals(mcpBase)
	Expect(redeemed.FlowHash).Equals(oauthas.HashToken("plain-flow"))
	var consent *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == handlers.MCPConsentCookie {
			consent = c
		}
	}
	Expect(consent).IsNotNil()
	Expect(consent.HttpOnly).IsTrue()
	claims, err := jwt.DecodeMCPConsent(consent.Value, mcpBase)
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(mock.AryaStark.ID)
}

// A resume link opened in another browser (no flow cookie) is refused: an
// insider cannot sign a colleague's browser in as themselves for consent.
func TestOAuthResume_RefusedWithoutTheFlowCookie(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	redeemCalled := false
	bus.AddHandler(func(ctx context.Context, c *cmd.RedeemOAuthSignInHandoff) error {
		redeemCalled = true
		return app.ErrNotFound
	})
	code, res := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).
		WithURL(mcpBase + "/oauth2/resume?code=abc").Execute(handlers.OAuthResume())
	Expect(code).Equals(http.StatusBadRequest)
	Expect(redeemCalled).IsFalse()
	for _, c := range res.Result().Cookies() {
		Expect(c.Name == handlers.MCPConsentCookie).IsFalse()
	}
}

func TestOAuthMCPCookies_HostPrefixed(t *testing.T) {
	RegisterT(t)
	Expect(handlers.MCPConsentCookie).Equals("__Host-mcp_consent")
	Expect(handlers.MCPFlowCookie).Equals("__Host-mcp_flow")
}

func TestOAuthResume_Refusals(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	bus.AddHandler(func(ctx context.Context, c *cmd.RedeemOAuthSignInHandoff) error { return app.ErrNotFound })
	code, res := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).
		AddCookie(handlers.MCPFlowCookie, "plain-flow").
		WithURL(mcpBase + "/oauth2/resume?code=used").Execute(handlers.OAuthResume())
	Expect(code).Equals(http.StatusBadRequest)
	for _, c := range res.Result().Cookies() {
		Expect(c.Name == handlers.MCPConsentCookie).IsFalse()
	}

	// only meaningful on an MCP-only address
	code, _ = mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).
		WithURL(boardBase + "/oauth2/resume?code=x").Execute(handlers.OAuthResume())
	Expect(code).Equals(http.StatusNotFound)
}

func TestOAuthAuthorize_ConsentShowsTheMCPAddress(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	mockOAuthClient()
	_, page := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeOn(mcpBase)).ExecuteAsPage(handlers.OAuthAuthorize())
	Expect(page.Data["connectingThrough"]).Equals("mcp.demo.test")

	_, page = mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeOn(boardBase)).ExecuteAsPage(handlers.OAuthAuthorize())
	Expect(page.Data["connectingThrough"]).IsNil()
}

func TestOAuthAuthorizeDecision_OnMCPOriginBindsCodeAndEndsConsent(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	saved := mockOAuthClient()
	code, res := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(mcpBase + "/_api/oauth2/authorize").ExecutePost(handlers.OAuthAuthorizeDecision(), approveBody(true))
	Expect(code).Equals(http.StatusOK)
	Expect(*saved).HasLen(1)
	Expect((*saved)[0].Origin).Equals(mcpBase)
	removed := false
	for _, c := range res.Result().Cookies() {
		if c.Name == handlers.MCPConsentCookie && c.MaxAge < 0 && c.Secure {
			removed = true
		}
	}
	Expect(removed).IsTrue()
}

func TestManageMCPPage_ListsPublicMCPAddresses(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	bus.AddHandler(func(ctx context.Context, q *query.ListOAuthClients) error { return nil })
	_, page := mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.JonSnow).
		WithURL(boardBase + "/admin/mcp").ExecuteAsPage(handlers.ManageMCPPage())
	Expect(page.Data["mcpUrl"]).Equals(boardBase + "/mcp")
	Expect(page.Data["publicMcpUrls"]).Equals([]any{mcpBase + "/mcp"})
}

// Signed out on a private site's own address: the normal sign-in, as before.
func TestOAuthAuthorize_SignedOutOnPrivateBoardGoesToSignIn(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()
	tenant := mcpTenantMinRole(enum.RoleVisitor)
	tenant.IsPrivate = true
	target := authorizeQuery(nil)
	code, res := mock.NewServer().OnTenant(tenant).WithURL(target).Execute(handlers.OAuthAuthorize())
	Expect(code).Equals(http.StatusTemporaryRedirect)
	u, _ := url.Parse(target)
	Expect(res.Header().Get("Location")).Equals("/signin?redirect=" + url.QueryEscape(u.RequestURI()))
}
