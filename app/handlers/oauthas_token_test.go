package handlers_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jsonq"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/oauthas"
)

const (
	testVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk" // RFC 7636 appendix B, matches testChallenge
	tokenURL     = "http://demo.test.fider.io/oauth2/token"
	testCode     = "the-plain-code"
	testRefresh  = "the-plain-refresh"
)

type tokenMocks struct {
	consumed *cmd.ConsumeOAuthCode
	saved    []*cmd.SaveOAuthRefreshToken
	rotated  *cmd.RotateOAuthRefreshToken
}

func mockTokenEndpoint(user *entity.User) *tokenMocks {
	m := &tokenMocks{}
	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthClient) error {
		if q.ClientID != testClientID {
			return app.ErrNotFound
		}
		q.Result = &entity.OAuthClient{ID: 1, ClientID: testClientID, Name: "Claude", RedirectURIs: []string{testRedirect}}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, c *cmd.ConsumeOAuthCode) error {
		m.consumed = c
		if c.CodeHash != oauthas.HashToken(testCode) || c.ClientID != testClientID {
			return app.ErrNotFound
		}
		c.Result = &entity.OAuthGrant{UserID: user.ID, ClientID: testClientID, Scope: "upvora", RedirectURI: testRedirect, CodeChallenge: testChallenge}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		if q.UserID != user.ID {
			return app.ErrNotFound
		}
		q.Result = user
		return nil
	})
	bus.AddHandler(func(ctx context.Context, c *cmd.SaveOAuthRefreshToken) error {
		m.saved = append(m.saved, c)
		return nil
	})
	bus.AddHandler(func(ctx context.Context, c *cmd.RotateOAuthRefreshToken) error {
		m.rotated = c
		if c.OldHash != oauthas.HashToken(testRefresh) || c.ClientID != testClientID {
			return app.ErrNotFound
		}
		c.Result = &entity.OAuthGrant{UserID: user.ID, ClientID: testClientID, Scope: "upvora:read", FamilyID: "fam"}
		return nil
	})
	return m
}

func postToken(tenant *entity.Tenant, form url.Values) (int, map[string]any) {
	code, res := mock.NewServer().OnTenant(tenant).WithURL(tokenURL).ExecutePostAsJSON(handlers.OAuthTokenEndpoint(), form.Encode())
	out := map[string]any{}
	for _, k := range []string{"access_token", "refresh_token", "token_type", "scope", "error"} {
		if res.Contains(k) {
			out[k] = res.String(k)
		}
	}
	if res.Contains("expires_in") {
		out["expires_in"] = res.Int32("expires_in")
	}
	return code, out
}

func codeForm(overrides map[string]string) url.Values {
	f := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {testCode},
		"redirect_uri":  {testRedirect},
		"client_id":     {testClientID},
		"code_verifier": {testVerifier},
	}
	for k, v := range overrides {
		f.Set(k, v)
	}
	return f
}

func TestOAuthToken_AuthorizationCode(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	user := *mock.AryaStark
	user.SecurityStamp = "stamp-1"
	m := mockTokenEndpoint(&user)

	code, out := postToken(mcpTenantMinRole(enum.RoleVisitor), codeForm(nil))

	Expect(code).Equals(http.StatusOK)
	Expect(out["token_type"]).Equals("Bearer")
	Expect(out["expires_in"]).Equals(3600)
	Expect(out["scope"]).Equals("upvora")
	Expect(m.saved).HasLen(1)
	Expect(m.saved[0].TokenHash).Equals(oauthas.HashToken(out["refresh_token"].(string)))
	Expect(m.saved[0].FromCodeHash).Equals(oauthas.HashToken(testCode)) // a replay of this code revokes the family

	claims, err := jwt.DecodeMCPAccessClaims(out["access_token"].(string), "http://demo.test.fider.io/mcp")
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(user.ID)
	Expect(claims.TenantID).Equals(mock.DemoTenant.ID)
	Expect(claims.ClientID).Equals(testClientID)
	Expect(claims.SecurityStamp).Equals("stamp-1")
}

func TestOAuthToken_AuthorizationCodeFailures(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	mockTokenEndpoint(mock.AryaStark)
	tenant := mcpTenantMinRole(enum.RoleVisitor)

	cases := []struct {
		form   url.Values
		status int
		err    string
	}{
		{codeForm(map[string]string{"code_verifier": testVerifier[:42] + "x"}), http.StatusBadRequest, "invalid_grant"},
		{codeForm(map[string]string{"redirect_uri": testRedirect + "/other"}), http.StatusBadRequest, "invalid_grant"},
		{codeForm(map[string]string{"code": "wrong"}), http.StatusBadRequest, "invalid_grant"},
		{codeForm(map[string]string{"client_id": "unknown"}), http.StatusUnauthorized, "invalid_client"},
		{codeForm(map[string]string{"grant_type": "password"}), http.StatusBadRequest, "unsupported_grant_type"},
	}
	for _, tc := range cases {
		code, out := postToken(tenant, tc.form)
		Expect(code).Equals(tc.status)
		Expect(out["error"]).Equals(tc.err)
		Expect(out["access_token"]).IsNil()
	}

	// role below the site's minimum at token time
	code, out := postToken(mcpTenantMinRole(enum.RoleCollaborator), codeForm(nil))
	Expect(code).Equals(http.StatusBadRequest)
	Expect(out["error"]).Equals("invalid_grant")

	// MCP off
	code, _ = postToken(mcpTenant(false, true), codeForm(nil))
	Expect(code).Equals(http.StatusNotFound)
}

func TestOAuthToken_BlockedUserGetsNothing(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	blocked := *mock.AryaStark
	blocked.Status = enum.UserBlocked
	mockTokenEndpoint(&blocked)

	code, out := postToken(mcpTenantMinRole(enum.RoleVisitor), codeForm(nil))
	Expect(code).Equals(http.StatusBadRequest)
	Expect(out["error"]).Equals("invalid_grant")
}

func TestOAuthToken_RefreshRotates(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	m := mockTokenEndpoint(mock.AryaStark)

	code, out := postToken(mcpTenantMinRole(enum.RoleVisitor), url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {testRefresh}, "client_id": {testClientID},
	})
	Expect(code).Equals(http.StatusOK)
	Expect(out["scope"]).Equals("upvora:read")
	Expect(m.rotated.NewHash).Equals(oauthas.HashToken(out["refresh_token"].(string)))
	Expect(out["refresh_token"] == testRefresh).IsFalse()

	code, out = postToken(mcpTenantMinRole(enum.RoleVisitor), url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {"stolen-or-reused"}, "client_id": {testClientID},
	})
	Expect(code).Equals(http.StatusBadRequest)
	Expect(out["error"]).Equals("invalid_grant")
}

func TestOAuthToken_RateLimited(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	mockTokenEndpoint(mock.AryaStark)
	last := 0
	for i := 0; i < 130; i++ {
		last, _ = postToken(mcpTenantMinRole(enum.RoleVisitor), codeForm(map[string]string{"code": "wrong"}))
	}
	Expect(last).Equals(http.StatusTooManyRequests)
	handlers.ResetOAuthRateLimits()
}

// Codes and refresh tokens are bound to the address they were issued on, and
// the access token's audience is that address's /mcp.
func TestOAuthToken_BoundToTheRequestAddress(t *testing.T) {
	RegisterT(t)
	withMCPOrigin(t)
	handlers.ResetOAuthRateLimits()
	m := mockTokenEndpoint(mock.AryaStark)
	post := func(base string, form url.Values) (int, *jsonq.Query) {
		return mock.NewSingleTenantServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).
			WithURL(base+"/oauth2/token").ExecutePostAsJSON(handlers.OAuthTokenEndpoint(), form.Encode())
	}

	code, res := post(mcpBase, codeForm(nil))
	Expect(code).Equals(http.StatusOK)
	Expect(m.consumed.Origin).Equals(mcpBase)
	Expect(m.consumed.AllowUnbound).IsFalse()
	Expect(m.saved[0].Origin).Equals(mcpBase)
	claims, err := jwt.DecodeMCPAccessClaims(res.String("access_token"), mcpBase+"/mcp")
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(mock.AryaStark.ID)

	code, _ = post(boardBase, codeForm(nil))
	Expect(code).Equals(http.StatusOK)
	Expect(m.consumed.Origin).Equals(boardBase)
	Expect(m.consumed.AllowUnbound).IsTrue()

	code, _ = post(mcpBase, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {testRefresh}, "client_id": {testClientID}})
	Expect(code).Equals(http.StatusOK)
	Expect(m.rotated.Origin).Equals(mcpBase)
	Expect(m.rotated.AllowUnbound).IsFalse()
}
