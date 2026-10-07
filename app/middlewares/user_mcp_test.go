package middlewares_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/middlewares"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/oauthas"
	"github.com/getfider/fider/app/pkg/web"
	jwtgo "github.com/golang-jwt/jwt/v4"
)

const mcpAudience = "http://demo.test.fider.io/mcp"

func mcpSite(enabled bool, minRole enum.Role) *entity.Tenant {
	t := *mock.DemoTenant
	t.MCPEnabled = enabled
	t.MCPMinRole = minRole
	return &t
}

func accessToken(user *entity.User, tenantID int, scope, aud, stamp string) string {
	token, _ := jwt.Encode(&jwt.MCPAccessClaims{
		UserID: user.ID, TenantID: tenantID, ClientID: "cid", Scope: scope, SecurityStamp: stamp,
		Metadata: jwt.Metadata{Audience: jwtgo.ClaimStrings{aud}, ExpiresAt: jwt.Time(time.Now().Add(time.Hour))},
	})
	return token
}

func mcpUser() *entity.User {
	u := *mock.AryaStark
	u.SecurityStamp = "stamp-1"
	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthClient) error {
		if q.ClientID != "cid" {
			return app.ErrNotFound
		}
		q.Result = &entity.OAuthClient{ClientID: "cid"}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		if q.UserID != u.ID {
			return app.ErrNotFound
		}
		q.Result = &u
		return nil
	})
	return &u
}

// asReplay marks the request as an in-process MCP tool replay, as the
// dispatcher does (a context value no external caller can set).
func asReplay(next web.HandlerFunc) web.HandlerFunc {
	return func(c *web.Context) error {
		c.Set(oauthas.ReplayCtxKey{}, true)
		return next(c)
	}
}

func runAs(tenant *entity.Tenant, method, url, bearer string, headers ...string) (int, http.Header, string) {
	return run(true, tenant, method, url, bearer, headers...)
}

func runDirect(tenant *entity.Tenant, method, url, bearer string, headers ...string) (int, http.Header, string) {
	return run(false, tenant, method, url, bearer, headers...)
}

func run(replay bool, tenant *entity.Tenant, method, url, bearer string, headers ...string) (int, http.Header, string) {
	server := mock.NewServer()
	if replay {
		server.Use(asReplay)
	}
	server.Use(middlewares.User())
	s := server.OnTenant(tenant).WithURL(url)
	if bearer != "" {
		s = s.AddHeader("Authorization", "Bearer "+bearer)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		s = s.AddHeader(headers[i], headers[i+1])
	}
	handler := func(c *web.Context) error {
		if c.User() == nil {
			return c.String(http.StatusOK, "anonymous")
		}
		return c.String(http.StatusOK, c.User().Name)
	}
	if method == "POST" {
		code, res := s.ExecutePost(handler, "{}")
		return code, res.Header(), res.Body.String()
	}
	code, res := s.Execute(handler)
	return code, res.Header(), res.Body.String()
}

func TestUser_MCPAccessToken_MemberAccepted(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok := accessToken(u, mock.DemoTenant.ID, "upvora", mcpAudience, "stamp-1")

	code, _, body := runAs(mcpSite(true, enum.RoleVisitor), "GET", "http://demo.test.fider.io/api/v1/posts", tok)
	Expect(code).Equals(http.StatusOK)
	Expect(body).Equals(u.Name)

	// impersonation is not available to access tokens
	code, _, body = runAs(mcpSite(true, enum.RoleVisitor), "GET", "http://demo.test.fider.io/api/v1/posts", tok, "X-Fider-UserID", "1")
	Expect(code).Equals(http.StatusOK)
	Expect(body).Equals(u.Name)
}

func TestUser_MCPAccessToken_Rejections(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	good := accessToken(u, mock.DemoTenant.ID, "upvora", mcpAudience, "stamp-1")

	cases := []struct {
		name   string
		tenant *entity.Tenant
		token  string
	}{
		{"stamp rotated (role change, block, sign-out)", mcpSite(true, enum.RoleVisitor), accessToken(u, mock.DemoTenant.ID, "upvora", mcpAudience, "old-stamp")},
		{"role below minimum", mcpSite(true, enum.RoleCollaborator), good},
		{"MCP turned off", mcpSite(false, enum.RoleVisitor), good},
		{"other site's audience", mcpSite(true, enum.RoleVisitor), accessToken(u, mock.DemoTenant.ID, "upvora", "http://other.test/mcp", "stamp-1")},
		{"other tenant id", mcpSite(true, enum.RoleVisitor), accessToken(u, 999, "upvora", mcpAudience, "stamp-1")},
		{"garbage jwt", mcpSite(true, enum.RoleVisitor), "aaa.bbb.ccc"},
	}
	for _, tc := range cases {
		code, _, _ := runAs(tc.tenant, "GET", "http://demo.test.fider.io/api/v1/posts", tc.token)
		if code != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", tc.name, code)
		}
	}
}

func TestUser_MCPAccessToken_ReadScope(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok := accessToken(u, mock.DemoTenant.ID, "upvora:read", mcpAudience, "stamp-1")

	code, _, _ := runAs(mcpSite(true, enum.RoleVisitor), "GET", "http://demo.test.fider.io/api/v1/posts", tok)
	Expect(code).Equals(http.StatusOK)
	code, _, _ = runAs(mcpSite(true, enum.RoleVisitor), "POST", "http://demo.test.fider.io/api/v1/posts", tok)
	Expect(code).Equals(http.StatusForbidden)
}

func TestUser_MCPAccessToken_NotAcceptedOutsideAPI(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok := accessToken(u, mock.DemoTenant.ID, "upvora", mcpAudience, "stamp-1")

	_, _, body := runAs(mcpSite(true, enum.RoleVisitor), "GET", "http://demo.test.fider.io/admin", tok)
	Expect(body).Equals("anonymous")
}

func TestUser_MCPEndpoint_ChallengesWithoutToken(t *testing.T) {
	RegisterT(t)
	mcpUser()
	code, headers, _ := runAs(mcpSite(true, enum.RoleVisitor), "POST", "http://demo.test.fider.io/mcp", "")
	Expect(code).Equals(http.StatusUnauthorized)
	Expect(headers.Get("WWW-Authenticate")).Equals(`Bearer resource_metadata="http://demo.test.fider.io/.well-known/oauth-protected-resource"`)
}

// A stale or foreign auth cookie must not shadow a valid MCP Bearer token.
func TestUser_MCPAccessToken_WinsOverCookie(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok := accessToken(u, mock.DemoTenant.ID, "upvora", mcpAudience, "stamp-1")

	code, _, body := runAs(mcpSite(true, enum.RoleVisitor), "GET", "http://demo.test.fider.io/api/v1/posts", tok, "Cookie", "auth=not-a-valid-jwt")
	Expect(code).Equals(http.StatusOK)
	Expect(body).Equals(u.Name)
}

// Every MCP message is a POST; a read-only token must still reach /mcp (write
// tools are hidden there, and each replayed call is checked with its own method).
func TestUser_MCPReadScope_CanUseMCPEndpoint(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok := accessToken(u, mock.DemoTenant.ID, "upvora:read", mcpAudience, "stamp-1")

	code, _, body := runAs(mcpSite(true, enum.RoleVisitor), "POST", "http://demo.test.fider.io/mcp", tok)
	Expect(code).Equals(http.StatusOK)
	Expect(body).Equals(u.Name)
}

// /mcp only takes Bearer tokens: a browser session must not list tools that
// would then run anonymously.
func TestUser_MCPEndpoint_RefusesCookieSessions(t *testing.T) {
	RegisterT(t)
	mcpUser()
	code, _, _ := runAs(mcpSite(true, enum.RoleVisitor), "POST", "http://demo.test.fider.io/mcp", "", "Cookie", "auth=whatever")
	Expect(code).Equals(http.StatusUnauthorized)
}

// An MCP token is for /mcp only: sent straight to /api/v1 it would skip the
// MCP budgets and audit, so it is refused there.
func TestUser_MCPAccessToken_RefusedOnDirectAPI(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok := accessToken(u, mock.DemoTenant.ID, "upvora", mcpAudience, "stamp-1")
	code, _, body := runDirect(mcpSite(true, enum.RoleVisitor), "GET", "http://demo.test.fider.io/api/v1/posts", tok)
	Expect(code).Equals(http.StatusUnauthorized)
	Expect(body == u.Name).IsFalse()
}

// Removing a client cuts off its access tokens at once, not after their hour.
func TestUser_MCPAccessToken_RemovedClientRefused(t *testing.T) {
	RegisterT(t)
	u := mcpUser()
	tok, _ := jwt.Encode(&jwt.MCPAccessClaims{
		UserID: u.ID, TenantID: mock.DemoTenant.ID, ClientID: "removed-client", Scope: "upvora", SecurityStamp: "stamp-1",
		Metadata: jwt.Metadata{Audience: jwtgo.ClaimStrings{mcpAudience}, ExpiresAt: jwt.Time(time.Now().Add(time.Hour))},
	})
	code, _, _ := runAs(mcpSite(true, enum.RoleVisitor), "POST", "http://demo.test.fider.io/mcp", tok)
	Expect(code).Equals(http.StatusUnauthorized)
}
