package cmd

import (
	"context"
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

	// direct REST call with the token
	direct, _ := http.NewRequest("GET", "http://demo.test.fider.io/api/v1/notifications", nil)
	direct.Header.Set("Authorization", "Bearer "+token)
	direct.RequestURI = direct.URL.RequestURI()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, direct)
	Expect(rec.Code).Equals(http.StatusOK)

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
