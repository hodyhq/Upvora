package handlers_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
)

const (
	testClientID    = "client-abc"
	testRedirect    = "https://claude.ai/api/mcp/auth_callback"
	testChallenge   = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	authorizeTarget = "http://demo.test.fider.io/oauth2/authorize"
)

func mockOAuthClient() *[]*cmd.SaveOAuthCode {
	saved := []*cmd.SaveOAuthCode{}
	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthClient) error {
		if q.ClientID != testClientID {
			return app.ErrNotFound
		}
		q.Result = &entity.OAuthClient{ID: 1, ClientID: testClientID, Name: "Claude", RedirectURIs: []string{testRedirect}}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, c *cmd.SaveOAuthCode) error {
		saved = append(saved, c)
		return nil
	})
	return &saved
}

func authorizeQuery(overrides map[string]string) string {
	v := url.Values{
		"response_type":         {"code"},
		"client_id":             {testClientID},
		"redirect_uri":          {testRedirect},
		"code_challenge":        {testChallenge},
		"code_challenge_method": {"S256"},
		"scope":                 {"upvora"},
		"state":                 {"st4te"},
	}
	for k, val := range overrides {
		if val == "" {
			v.Del(k)
		} else {
			v.Set(k, val)
		}
	}
	return authorizeTarget + "?" + v.Encode()
}

func mcpTenantMinRole(role enum.Role) *entity.Tenant {
	t := mcpTenant(true, true)
	t.MCPMinRole = role
	return t
}

func TestOAuthAuthorize_RendersConsent(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()

	code, page := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeQuery(nil)).ExecuteAsPage(handlers.OAuthAuthorize())

	Expect(code).Equals(http.StatusOK)
	Expect(page.Page).Equals("OAuthConsent/OAuthConsent.page")
	Expect(page.Data["clientName"]).Equals("Claude")
	Expect(page.Data["redirectHost"]).Equals("claude.ai")
	Expect(page.Data["scope"]).Equals("upvora")
}

// Before the client and redirect_uri are verified, errors are shown on our page,
// never redirected (no open redirect).
func TestOAuthAuthorize_UnverifiedRedirectNeverFollowed(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()

	for _, o := range []map[string]string{
		{"client_id": "unknown"},
		{"redirect_uri": "https://evil.example/cb"},
		{"redirect_uri": testRedirect + "/extra"},
		{"redirect_uri": ""},
	} {
		code, page := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
			WithURL(authorizeQuery(o)).ExecuteAsPage(handlers.OAuthAuthorize())
		Expect(code).Equals(http.StatusBadRequest)
		Expect(page.Page).Equals("OAuthConsent/OAuthConsent.page")
		Expect(page.Data["error"] != nil).IsTrue()
	}
}

// Once the redirect is verified, protocol errors go back to the client.
func TestOAuthAuthorize_ProtocolErrorsRedirect(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()

	cases := map[string]map[string]string{
		"unsupported_response_type": {"response_type": "token"},
		"invalid_request":           {"code_challenge_method": "plain"},
		"invalid_scope":             {"scope": "admin"},
	}
	for want, o := range cases {
		code, res := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
			WithURL(authorizeQuery(o)).Execute(handlers.OAuthAuthorize())
		Expect(code).Equals(http.StatusTemporaryRedirect)
		loc := res.Header().Get("Location")
		Expect(strings.HasPrefix(loc, testRedirect+"?")).IsTrue()
		Expect(loc).ContainsSubstring("error=" + want)
		Expect(loc).ContainsSubstring("state=st4te")
	}

	// missing PKCE challenge
	_, res := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeQuery(map[string]string{"code_challenge": ""})).Execute(handlers.OAuthAuthorize())
	Expect(res.Header().Get("Location")).ContainsSubstring("error=invalid_request")
}

func TestOAuthAuthorize_RoleBelowMinimumIsDenied(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()

	code, page := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleCollaborator)).AsUser(mock.AryaStark).
		WithURL(authorizeQuery(nil)).ExecuteAsPage(handlers.OAuthAuthorize())
	Expect(code).Equals(http.StatusForbidden)
	Expect(page.Data["error"]).Equals("Your role on this site is not allowed to connect MCP clients.")
}

func TestOAuthAuthorize_DisabledWhenMCPOff(t *testing.T) {
	RegisterT(t)
	mockOAuthClient()
	code, _ := mock.NewServer().OnTenant(mcpTenant(false, true)).AsUser(mock.AryaStark).
		WithURL(authorizeQuery(nil)).Execute(handlers.OAuthAuthorize())
	Expect(code).Equals(http.StatusNotFound)
}

func approveBody(approve bool) string {
	a := "false"
	if approve {
		a = "true"
	}
	return `{"clientId":"` + testClientID + `","redirectUri":"` + testRedirect + `","codeChallenge":"` + testChallenge +
		`","codeChallengeMethod":"S256","scope":"upvora","state":"st4te","approve":` + a + `}`
}

func TestOAuthAuthorizeDecision_ApproveIssuesBoundCode(t *testing.T) {
	RegisterT(t)
	saved := mockOAuthClient()

	code, res := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeTarget).ExecutePostAsJSON(handlers.OAuthAuthorizeDecision(), approveBody(true))

	Expect(code).Equals(http.StatusOK)
	loc, _ := url.Parse(res.String("redirect"))
	Expect(loc.Scheme + "://" + loc.Host + loc.Path).Equals(testRedirect)
	Expect(loc.Query().Get("state")).Equals("st4te")
	Expect(loc.Query().Get("iss")).Equals("http://demo.test.fider.io")
	Expect(len(loc.Query().Get("code")) >= 43).IsTrue()

	Expect(*saved).HasLen(1)
	s := (*saved)[0]
	Expect(s.UserID).Equals(mock.AryaStark.ID)
	Expect(s.ClientID).Equals(testClientID)
	Expect(s.RedirectURI).Equals(testRedirect)
	Expect(s.CodeChallenge).Equals(testChallenge)
	Expect(s.CodeHash == loc.Query().Get("code")).IsFalse() // only the hash is stored
}

func TestOAuthAuthorizeDecision_DenyAndRevalidation(t *testing.T) {
	RegisterT(t)
	saved := mockOAuthClient()

	_, res := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeTarget).ExecutePostAsJSON(handlers.OAuthAuthorizeDecision(), approveBody(false))
	Expect(res.String("redirect")).ContainsSubstring("error=access_denied")
	Expect(*saved).HasLen(0)

	// a tampered redirect in the POST is not trusted
	bad := strings.Replace(approveBody(true), testRedirect, "https://evil.example/cb", 1)
	code, _ := mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleVisitor)).AsUser(mock.AryaStark).
		WithURL(authorizeTarget).ExecutePost(handlers.OAuthAuthorizeDecision(), bad)
	Expect(code).Equals(http.StatusBadRequest)
	Expect(*saved).HasLen(0)

	// the gate is re-checked on approval
	code, _ = mock.NewServer().OnTenant(mcpTenantMinRole(enum.RoleAdministrator)).AsUser(mock.AryaStark).
		WithURL(authorizeTarget).ExecutePost(handlers.OAuthAuthorizeDecision(), approveBody(true))
	Expect(code).Equals(http.StatusForbidden)
	Expect(*saved).HasLen(0)
}
