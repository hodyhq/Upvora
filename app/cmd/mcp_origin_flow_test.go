package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/getfider/fider/app/mcpserver"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
	"github.com/getfider/fider/app/services/sqlstore/postgres"
)

const (
	flowBoard     = "https://board.demo.test"
	flowMCP       = "https://mcp.demo.test"
	flowRedirect  = "https://claude.ai/api/mcp/auth_callback"
	flowVerifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	flowChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
)

type flowCall struct {
	method, target, body, contentType string
	cookies                           []*http.Cookie
	bearer                            string
}

func serve(engine *web.Engine, call flowCall) *httptest.ResponseRecorder {
	var body io.Reader
	if call.body != "" {
		body = strings.NewReader(call.body)
	}
	req := httptest.NewRequest(call.method, call.target, body)
	req.RequestURI = req.URL.RequestURI()
	if call.contentType != "" {
		req.Header.Set("Content-Type", call.contentType)
	}
	for _, c := range call.cookies {
		req.AddCookie(c)
	}
	if call.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+call.bearer)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// The whole connection for a board that stays internal: the client and its
// browser only ever use the MCP-only address, signing in happens on the board,
// and the token works on the MCP-only address alone.
func TestMCPOrigin_FullConnectionThroughRealEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("needs the test database")
	}
	RegisterT(t)
	mode, base := env.Config.HostMode, env.Config.BaseURL
	env.Config.HostMode, env.Config.BaseURL, env.Config.MCPOrigins = "single", flowBoard, []string{flowMCP}
	defer func() { env.Config.HostMode, env.Config.BaseURL, env.Config.MCPOrigins = mode, base, nil }()
	bus.Init(postgres.Service{})
	// pages list sign-in providers; that service is not under test here
	bus.AddHandler(func(ctx context.Context, q *query.ListActiveOAuthProviders) error { return nil })
	dbx.Seed()
	defer dbx.Seed()

	trx, err := dbx.BeginTx(context.Background())
	Expect(err).IsNil()
	_, err = trx.Execute("UPDATE tenants SET mcp_enabled = true, mcp_min_role = 1 WHERE id = 1")
	Expect(err).IsNil()
	_, err = trx.Execute("INSERT INTO oauth_clients (tenant_id, client_id, name, redirect_uris) VALUES (1, 'cid', 'Claude', '{" + flowRedirect + "}')")
	Expect(err).IsNil()
	var stamp, name string
	Expect(trx.Scalar(&stamp, "SELECT COALESCE(security_stamp, '') FROM users WHERE id = 2")).IsNil()
	Expect(trx.Scalar(&name, "SELECT name FROM users WHERE id = 2")).IsNil()
	Expect(trx.Commit()).IsNil()
	engine := routes(web.New())

	// The board itself is not served on the MCP-only address.
	for _, p := range []string{"/", "/admin", "/posts/1", "/signin", "/api/v1/posts", "/oauth2/continue"} {
		Expect(serve(engine, flowCall{method: "GET", target: flowMCP + p}).Code).Equals(http.StatusNotFound)
	}

	// 1. The client sends the browser to authorize on the MCP-only address.
	q := url.Values{"response_type": {"code"}, "client_id": {"cid"}, "redirect_uri": {flowRedirect},
		"code_challenge": {flowChallenge}, "code_challenge_method": {"S256"}, "scope": {"upvora"}, "state": {"st"}}
	rec := serve(engine, flowCall{method: "GET", target: flowMCP + "/oauth2/authorize?" + q.Encode()})
	Expect(rec.Code).Equals(http.StatusTemporaryRedirect)
	toBoard := rec.Header().Get("Location")
	Expect(strings.HasPrefix(toBoard, flowBoard+"/oauth2/continue?request=")).IsTrue()

	// 2. On the board, signed in with its normal session, it hands back.
	session, _ := jwt.Encode(jwt.FiderClaims{UserID: 2, UserName: name, SecurityStamp: stamp})
	rec = serve(engine, flowCall{method: "GET", target: toBoard, cookies: []*http.Cookie{{Name: web.CookieAuthName, Value: session}}})
	Expect(rec.Code).Equals(http.StatusTemporaryRedirect)
	toResume := rec.Header().Get("Location")
	Expect(strings.HasPrefix(toResume, flowMCP+"/oauth2/resume?code=")).IsTrue()

	// 3. Back on the MCP-only address: a consent-step cookie, then authorize again.
	rec = serve(engine, flowCall{method: "GET", target: toResume})
	Expect(rec.Code).Equals(http.StatusTemporaryRedirect)
	Expect(rec.Header().Get("Location")).Equals("/oauth2/authorize?" + q.Encode())
	var consent []*http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == web.CookieMCPConsentName {
			consent = append(consent, c)
		}
	}
	Expect(consent).HasLen(1)
	Expect(consent[0].HttpOnly).IsTrue()
	Expect(consent[0].Secure).IsTrue()
	// the handoff code is single-use
	reused := serve(engine, flowCall{method: "GET", target: toResume})
	Expect(reused.Code).Equals(http.StatusBadRequest)

	// 4. The consent page renders there, self-contained on that address.
	rec = serve(engine, flowCall{method: "GET", target: flowMCP + "/oauth2/authorize?" + q.Encode(), cookies: consent})
	Expect(rec.Code).Equals(http.StatusOK)
	page := rec.Body.String()
	Expect(page).ContainsSubstring(`"connectingThrough":"mcp.demo.test"`)
	Expect(page).ContainsSubstring(`"assetsURL":"https://mcp.demo.test"`)
	Expect(strings.Contains(page, "board.demo.test")).IsFalse()

	// The person approves; the code comes back for the client.
	approve, _ := json.Marshal(map[string]any{"clientId": "cid", "redirectUri": flowRedirect, "codeChallenge": flowChallenge,
		"codeChallengeMethod": "S256", "scope": "upvora", "state": "st", "approve": true})
	rec = serve(engine, flowCall{method: "POST", target: flowMCP + "/_api/oauth2/authorize", body: string(approve),
		contentType: "application/json", cookies: consent})
	Expect(rec.Code).Equals(http.StatusOK)
	var decision struct{ Redirect string }
	Expect(json.Unmarshal(rec.Body.Bytes(), &decision)).IsNil()
	back, _ := url.Parse(decision.Redirect)
	Expect(back.Query().Get("iss")).Equals(flowMCP)
	code := back.Query().Get("code")

	// The code is bound to the MCP-only address: refused on the board, not burned.
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {"cid"},
		"redirect_uri": {flowRedirect}, "code_verifier": {flowVerifier}}
	rec = serve(engine, flowCall{method: "POST", target: flowBoard + "/oauth2/token", body: form.Encode(), contentType: "application/x-www-form-urlencoded"})
	Expect(rec.Code).Equals(http.StatusBadRequest)

	// 5. The client (server to server) exchanges it on the MCP-only address.
	rec = serve(engine, flowCall{method: "POST", target: flowMCP + "/oauth2/token", body: form.Encode(), contentType: "application/x-www-form-urlencoded"})
	Expect(rec.Code).Equals(http.StatusOK)
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	Expect(json.Unmarshal(rec.Body.Bytes(), &tokens)).IsNil()

	// 6. The token works on the MCP-only address, and only there.
	mcpReq, _ := http.NewRequest("POST", flowMCP+"/mcp", nil)
	mcpReq.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	var tool mcpserver.Tool
	for _, c := range mcpserver.Catalog {
		if c.Name == "upvora_notifications_list" {
			tool = c
		}
	}
	res := mcpserver.Dispatch(engine, mcpReq, tool, map[string]any{})
	if res.IsError {
		t.Fatalf("tool call on the MCP-only address failed: %s", res.Text)
	}
	init := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	Expect(serve(engine, flowCall{method: "POST", target: flowBoard + "/mcp", body: init, contentType: "application/json", bearer: tokens.AccessToken}).Code).Equals(http.StatusUnauthorized)

	// 7. Refresh works on the MCP-only address only.
	refresh := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.RefreshToken}, "client_id": {"cid"}}
	rec = serve(engine, flowCall{method: "POST", target: flowBoard + "/oauth2/token", body: refresh.Encode(), contentType: "application/x-www-form-urlencoded"})
	Expect(rec.Code).Equals(http.StatusBadRequest)
	rec = serve(engine, flowCall{method: "POST", target: flowMCP + "/oauth2/token", body: refresh.Encode(), contentType: "application/x-www-form-urlencoded"})
	Expect(rec.Code).Equals(http.StatusOK)
}
