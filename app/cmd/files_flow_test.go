package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getfider/fider/app/mcpserver"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
	blobsql "github.com/getfider/fider/app/services/blob/sql"
	"github.com/getfider/fider/app/services/sqlstore/postgres"
	jwtgo "github.com/golang-jwt/jwt/v4"
)

// An assistant attaches a Markdown file to a new idea over MCP; the file is
// stored apart from images and served only to those who can see the idea.
func TestFiles_AttachOverMCPAndServeByVisibility(t *testing.T) {
	if testing.Short() {
		t.Skip("needs the test database")
	}
	RegisterT(t)
	blobType := env.Config.BlobStorage.Type
	env.Config.BlobStorage.Type = "sql"
	defer func() { env.Config.BlobStorage.Type = blobType }()
	bus.Init(postgres.Service{}, blobsql.Service{})
	bus.AddHandler(func(ctx context.Context, q *query.ListActiveOAuthProviders) error { return nil })
	dbx.Seed()
	defer dbx.Seed()

	trx, err := dbx.BeginTx(context.Background())
	Expect(err).IsNil()
	_, err = trx.Execute("UPDATE tenants SET mcp_enabled = true, mcp_min_role = 1, file_attachments_enabled = true WHERE id = 1")
	Expect(err).IsNil()
	_, err = trx.Execute("INSERT INTO oauth_clients (tenant_id, client_id, name, redirect_uris) VALUES (1, 'cid', 'test', '{https://x.example/cb}')")
	Expect(err).IsNil()
	var stamp string
	Expect(trx.Scalar(&stamp, "SELECT COALESCE(security_stamp, '') FROM users WHERE id = 2")).IsNil()
	Expect(trx.Commit()).IsNil()

	token, _ := jwt.Encode(&jwt.MCPAccessClaims{
		UserID: 2, TenantID: 1, ClientID: "cid", Scope: "upvora", SecurityStamp: stamp,
		Metadata: jwt.Metadata{Audience: jwtgo.ClaimStrings{"http://demo.test.fider.io/mcp"}, ExpiresAt: jwt.Time(time.Now().Add(time.Hour))},
	})
	engine := routes(web.New())

	orig, _ := http.NewRequest("POST", "http://demo.test.fider.io/mcp", nil)
	orig.Header.Set("Authorization", "Bearer "+token)
	var create mcpserver.Tool
	for _, c := range mcpserver.Catalog {
		if c.Name == "upvora_ideas_create" {
			create = c
		}
	}
	res := mcpserver.Dispatch(engine, orig, create, map[string]any{
		"title":       "An idea with a Markdown plan attached",
		"attachments": []any{map[string]any{"upload": map[string]any{"fileName": "plan.md", "text": "# Plan\n\n<script>alert(1)</script>"}}},
	})
	if res.IsError {
		t.Fatalf("create failed: %s", res.Text)
	}

	trx, _ = dbx.BeginTx(context.Background())
	var key string
	var postID int
	Expect(trx.Scalar(&key, "SELECT attachment_bkey FROM attachments WHERE tenant_id = 1 ORDER BY id DESC LIMIT 1")).IsNil()
	Expect(trx.Scalar(&postID, "SELECT post_id FROM attachments WHERE attachment_bkey = $1", key)).IsNil()
	var description string
	Expect(trx.Scalar(&description, "SELECT description FROM posts WHERE id = $1", postID)).IsNil()
	trx.MustRollback()
	Expect(strings.HasPrefix(key, "files/")).IsTrue()
	Expect(strings.HasSuffix(key, "-plan.md")).IsTrue()
	Expect(strings.Contains(description, key)).IsFalse() // listed, not embedded as an image

	get := func(path string, user int) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "http://demo.test.fider.io"+path, nil)
		req.RequestURI = req.URL.RequestURI()
		if user > 0 {
			session, _ := jwt.Encode(jwt.FiderClaims{UserID: user, UserName: "x", SecurityStamp: stamp})
			req.AddCookie(&http.Cookie{Name: web.CookieAuthName, Value: session})
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/static/files/"+key, 0) // public idea: anyone who can see it
	Expect(rec.Code).Equals(http.StatusOK)
	Expect(rec.Header().Get("Content-Type")).Equals("text/plain; charset=utf-8")
	Expect(rec.Header().Get("Content-Security-Policy")).Equals("default-src 'none'; sandbox")
	Expect(rec.Body.String()).ContainsSubstring("# Plan")
	Expect(get("/static/images/"+key, 0).Code).Equals(http.StatusNotFound)

	trx, _ = dbx.BeginTx(context.Background())
	_, err = trx.Execute("UPDATE posts SET is_private = true WHERE id = $1", postID)
	Expect(err).IsNil()
	Expect(trx.Commit()).IsNil()
	Expect(get("/static/files/"+key, 0).Code).Equals(http.StatusNotFound) // anonymous
	Expect(get("/static/files/"+key, 2).Code).Equals(http.StatusOK)       // the author
}
