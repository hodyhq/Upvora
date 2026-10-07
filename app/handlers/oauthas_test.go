package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

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

func mcpTenant(enabled, dcr bool) *entity.Tenant {
	t := *mock.DemoTenant
	t.MCPEnabled = enabled
	t.MCPDCREnabled = dcr
	t.MCPMinRole = enum.RoleAdministrator
	return &t
}

func TestOAuthASMetadata(t *testing.T) {
	RegisterT(t)
	code, res := mock.NewServer().OnTenant(mcpTenant(true, true)).WithURL("http://demo.test.fider.io:3000/.well-known/oauth-authorization-server").ExecuteAsJSON(handlers.OAuthAuthorizationServerMetadata())

	Expect(code).Equals(http.StatusOK)
	Expect(res.String("issuer")).Equals("http://demo.test.fider.io:3000")
	Expect(res.String("authorization_endpoint")).Equals("http://demo.test.fider.io:3000/oauth2/authorize")
	Expect(res.String("token_endpoint")).Equals("http://demo.test.fider.io:3000/oauth2/token")
	Expect(res.String("registration_endpoint")).Equals("http://demo.test.fider.io:3000/oauth2/register")
	Expect(res.Strings("code_challenge_methods_supported")).Equals([]string{"S256"})
	Expect(res.Strings("token_endpoint_auth_methods_supported")).Equals([]string{"none"})

	// no registration endpoint advertised when DCR is off
	_, res = mock.NewServer().OnTenant(mcpTenant(true, false)).ExecuteAsJSON(handlers.OAuthAuthorizationServerMetadata())
	Expect(res.Contains("registration_endpoint")).IsFalse()
}

func TestOAuthProtectedResourceMetadata(t *testing.T) {
	RegisterT(t)
	code, res := mock.NewServer().OnTenant(mcpTenant(true, true)).WithURL("http://demo.test.fider.io:3000/.well-known/oauth-protected-resource").ExecuteAsJSON(handlers.OAuthProtectedResourceMetadata())

	Expect(code).Equals(http.StatusOK)
	Expect(res.String("resource")).Equals("http://demo.test.fider.io:3000/mcp")
	Expect(res.Strings("authorization_servers")).Equals([]string{"http://demo.test.fider.io:3000"})
}

func TestOAuthEndpoints_DisabledWhenMCPOff(t *testing.T) {
	RegisterT(t)
	code, _ := mock.NewServer().OnTenant(mcpTenant(false, true)).Execute(handlers.OAuthAuthorizationServerMetadata())
	Expect(code).Equals(http.StatusNotFound)
	code, _ = mock.NewServer().OnTenant(mcpTenant(false, true)).Execute(handlers.OAuthProtectedResourceMetadata())
	Expect(code).Equals(http.StatusNotFound)
	code, _ = mock.NewServer().OnTenant(mcpTenant(false, true)).ExecutePost(handlers.OAuthRegister(), `{"redirect_uris":["https://claude.ai/cb"]}`)
	Expect(code).Equals(http.StatusNotFound)
}

func TestOAuthRegister(t *testing.T) {
	RegisterT(t)
	var saved *cmd.RegisterOAuthClient
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterOAuthClient) error {
		saved = c
		c.Result = &entity.OAuthClient{ID: 1, ClientID: "cid-123", Name: c.Name, RedirectURIs: c.RedirectURIs}
		return nil
	})

	code, res := mock.NewServer().OnTenant(mcpTenant(true, true)).
		ExecutePostAsJSON(handlers.OAuthRegister(), `{"client_name":"Claude","redirect_uris":["https://claude.ai/api/mcp/auth_callback"],"token_endpoint_auth_method":"none"}`)

	Expect(code).Equals(http.StatusCreated)
	Expect(res.String("client_id")).Equals("cid-123")
	Expect(res.String("token_endpoint_auth_method")).Equals("none")
	Expect(saved.CreatedByAdmin).IsFalse()
}

func TestOAuthRegister_Rejections(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterOAuthClient) error {
		c.Result = &entity.OAuthClient{ClientID: "x"}
		return nil
	})

	cases := []struct {
		tenant *entity.Tenant
		body   string
		status int
	}{
		{mcpTenant(true, false), `{"redirect_uris":["https://claude.ai/cb"]}`, http.StatusForbidden},
		{mcpTenant(true, true), `{"redirect_uris":[]}`, http.StatusBadRequest},
		{mcpTenant(true, true), `{"redirect_uris":["javascript:alert(1)"]}`, http.StatusBadRequest},
		{mcpTenant(true, true), `{"redirect_uris":["http://evil.example/cb"]}`, http.StatusBadRequest},
		{mcpTenant(true, true), `{"redirect_uris":["https://claude.ai/cb"],"token_endpoint_auth_method":"client_secret_basic"}`, http.StatusBadRequest},
		{mcpTenant(true, true), `not json`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		code, _ := mock.NewServer().OnTenant(tc.tenant).ExecutePost(handlers.OAuthRegister(), tc.body)
		Expect(code).Equals(tc.status)
	}
}

func TestOAuthRegister_RateLimited(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterOAuthClient) error {
		c.Result = &entity.OAuthClient{ClientID: "x"}
		return nil
	})

	last := 0
	for i := 0; i < 25; i++ {
		last, _ = mock.NewServer().OnTenant(mcpTenant(true, true)).AddHeader("CF-Connecting-IP", "203.0.113.50").
			ExecutePost(handlers.OAuthRegister(), `{"redirect_uris":["https://claude.ai/cb"]}`)
	}
	Expect(last).Equals(http.StatusTooManyRequests)
	handlers.ResetOAuthRateLimits()
}

func TestMCPClients_AdminCreateListDelete(t *testing.T) {
	RegisterT(t)
	var created *cmd.RegisterOAuthClient
	deleted := 0
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterOAuthClient) error {
		created = c
		c.Result = &entity.OAuthClient{ID: 5, ClientID: "cid-new", Name: c.Name, RedirectURIs: c.RedirectURIs, CreatedByAdmin: c.CreatedByAdmin}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.ListOAuthClients) error {
		q.Result = []*entity.OAuthClient{{ID: 5, ClientID: "cid-new"}}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, c *cmd.DeleteOAuthClient) error {
		deleted = c.ID
		return nil
	})

	code, res := mock.NewServer().OnTenant(mcpTenant(true, false)).AsUser(mock.JonSnow).
		ExecutePostAsJSON(handlers.CreateMCPClient(), `{"name":"Team Claude","redirectUris":["https://claude.ai/api/mcp/auth_callback"]}`)
	Expect(code).Equals(http.StatusOK)
	Expect(res.String("clientId")).Equals("cid-new")
	Expect(created.CreatedByAdmin).IsTrue()

	code, _ = mock.NewServer().OnTenant(mcpTenant(true, false)).AsUser(mock.JonSnow).
		ExecutePost(handlers.CreateMCPClient(), `{"name":"Bad","redirectUris":["javascript:alert(1)"]}`)
	Expect(code).Equals(http.StatusBadRequest)

	code, list := mock.NewServer().OnTenant(mcpTenant(true, false)).AsUser(mock.JonSnow).ExecuteAsJSON(handlers.ListMCPClients())
	Expect(code).Equals(http.StatusOK)
	Expect(list.ArrayLength()).Equals(1)

	code, _ = mock.NewServer().OnTenant(mcpTenant(true, false)).AsUser(mock.JonSnow).AddParam("id", 5).Execute(handlers.DeleteMCPClient())
	Expect(code).Equals(http.StatusOK)
	Expect(deleted).Equals(5)
}

// Long multi-byte names are cut on a character boundary, never mid-character.
func TestOAuthRegister_NameTruncatedByRunes(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	var saved *cmd.RegisterOAuthClient
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterOAuthClient) error {
		saved = c
		c.Result = &entity.OAuthClient{ClientID: "x", Name: c.Name}
		return nil
	})
	name := strings.Repeat("é", 150)
	mock.NewServer().OnTenant(mcpTenant(true, true)).
		ExecutePost(handlers.OAuthRegister(), `{"client_name":"`+name+`","redirect_uris":["https://claude.ai/cb"]}`)
	Expect(utf8.ValidString(saved.Name)).IsTrue()
	Expect(utf8.RuneCountInString(saved.Name)).Equals(100)
}

// The token endpoint has a per-site cap on top of the (forgeable) per-IP one.
func TestOAuthToken_PerSiteCap(t *testing.T) {
	RegisterT(t)
	handlers.ResetOAuthRateLimits()
	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthClient) error { return app.ErrNotFound })
	last := 0
	for i := 0; i < 1100; i++ {
		last, _ = mock.NewServer().OnTenant(mcpTenant(true, true)).AddHeader("CF-Connecting-IP", fmt.Sprintf("198.51.%d.%d", i/250, i%250)).
			ExecutePost(handlers.OAuthTokenEndpoint(), "grant_type=authorization_code&client_id=x")
	}
	Expect(last).Equals(http.StatusTooManyRequests)
	handlers.ResetOAuthRateLimits()
}
