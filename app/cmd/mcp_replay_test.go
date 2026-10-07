package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getfider/fider/app/mcpserver"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
	"github.com/getfider/fider/app/services/sqlstore/postgres"
	jwtgo "github.com/golang-jwt/jwt/v4"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// End to end through the real engine and store: an MCP tool call replayed
// in-process must authenticate as the token's user.
func TestMCPReplay_AuthenticatesThroughRealEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("needs the test database")
	}
	RegisterT(t)
	bus.Init(postgres.Service{})
	dbx.Seed()
	defer dbx.Seed()

	ctx := context.Background()
	trx, err := dbx.BeginTx(ctx)
	Expect(err).IsNil()
	_, err = trx.Execute("UPDATE tenants SET mcp_enabled = true, mcp_min_role = 1 WHERE id = 1")
	Expect(err).IsNil()
	var stamp string
	Expect(trx.Scalar(&stamp, "SELECT COALESCE(security_stamp, '') FROM users WHERE id = 2")).IsNil()
	Expect(trx.Commit()).IsNil()

	token, _ := jwt.Encode(&jwt.MCPAccessClaims{
		UserID: 2, TenantID: 1, ClientID: "cid", Scope: "upvora", SecurityStamp: stamp,
		Metadata: jwt.Metadata{Audience: jwtgo.ClaimStrings{"http://demo.test.fider.io/mcp"}, ExpiresAt: jwt.Time(time.Now().Add(time.Hour))},
	})
	engine := routes(web.New())

	// the token is for /mcp: sent straight to the REST API it is refused
	direct, _ := http.NewRequest("GET", "http://demo.test.fider.io/api/v1/notifications", nil)
	direct.Header.Set("Authorization", "Bearer "+token)
	direct.RequestURI = direct.URL.RequestURI()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, direct)
	Expect(rec.Code).Equals(http.StatusUnauthorized)

	// the same call replayed as an MCP tool
	orig, _ := http.NewRequest("POST", "http://demo.test.fider.io/mcp", nil)
	orig.Header.Set("Authorization", "Bearer "+token)
	var tool mcpserver.Tool
	for _, c := range mcpserver.Catalog {
		if c.Name == "upvora_notifications_list" {
			tool = c
		}
	}
	res := mcpserver.Dispatch(engine, orig, tool, map[string]any{})
	if res.IsError {
		t.Fatalf("replay failed: %s", res.Text)
	}
}

// More concurrent tool calls than the DB pool (default 4) must not deadlock:
// the /mcp request may not hold its transaction while the replay needs one.
func TestMCPReplay_ConcurrentCallsDoNotExhaustThePool(t *testing.T) {
	if testing.Short() {
		t.Skip("needs the test database")
	}
	RegisterT(t)
	bus.Init(postgres.Service{})
	dbx.Seed()
	defer dbx.Seed()

	ctx := context.Background()
	trx, _ := dbx.BeginTx(ctx)
	_, _ = trx.Execute("UPDATE tenants SET mcp_enabled = true, mcp_min_role = 1 WHERE id = 1")
	var stamp string
	_ = trx.Scalar(&stamp, "SELECT COALESCE(security_stamp, '') FROM users WHERE id = 2")
	Expect(trx.Commit()).IsNil()
	token, _ := jwt.Encode(&jwt.MCPAccessClaims{
		UserID: 2, TenantID: 1, ClientID: "cid", Scope: "upvora", SecurityStamp: stamp,
		Metadata: jwt.Metadata{Audience: jwtgo.ClaimStrings{"http://demo.test.fider.io/mcp"}, ExpiresAt: jwt.Time(time.Now().Add(time.Hour))},
	})

	ts := httptest.NewServer(routes(web.New()))
	defer ts.Close()
	// route every request to the demo tenant's host
	client := &http.Client{Transport: hostRewrite{token: token}}

	const calls = 6
	errs := make(chan error, calls)
	deadline, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for i := 0; i < calls; i++ {
		go func() {
			c := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "1"}, nil)
			s, err := c.Connect(deadline, &sdk.StreamableClientTransport{Endpoint: ts.URL + "/mcp", HTTPClient: client}, nil)
			if err != nil {
				errs <- err
				return
			}
			defer s.Close()
			res, err := s.CallTool(deadline, &sdk.CallToolParams{Name: "upvora_notifications_list", Arguments: map[string]any{}})
			if err == nil && res.IsError {
				err = fmt.Errorf("tool error: %v", res.Content)
			}
			errs <- err
		}()
	}
	for i := 0; i < calls; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent call failed: %v", err)
		}
	}
}

type hostRewrite struct{ token string }

func (h hostRewrite) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Host = "demo.test.fider.io"
	r.Header.Set("Authorization", "Bearer "+h.token)
	return http.DefaultTransport.RoundTrip(r)
}
