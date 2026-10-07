package mcpserver_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/getfider/fider/app/mcpserver"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/oauthas"
)

// echoEngine records what a replayed request looks like.
type echoEngine struct{ status int }

func (e echoEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	status := e.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"method": r.Method, "path": r.URL.EscapedPath(), "query": r.URL.RawQuery, "body": string(body),
		"auth": r.Header.Get("Authorization"), "host": r.Host, "proto": r.Header.Get("X-Forwarded-Proto"),
	})
}

func origRequest() *http.Request {
	r, _ := http.NewRequest("POST", "https://demo.test/mcp", nil)
	r.Host = "demo.test"
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("X-Forwarded-Proto", "https")
	return r
}

func decode(t *testing.T, s string) map[string]string {
	out := map[string]string{}
	Expect(json.Unmarshal([]byte(s), &out)).IsNil()
	return out
}

var commentTool = mcpserver.Tool{
	Name: "upvora_comments_update", Method: "PUT", Path: "/api/v1/posts/{number}/comments/{id}",
	MinRole: enum.RoleVisitor,
	Fields: []mcpserver.Field{
		{Name: "number", Type: "integer", In: "path", Required: true},
		{Name: "id", Type: "integer", In: "path", Required: true},
		{Name: "content", Type: "string", In: "body", Required: true},
	},
}

func TestDispatch_PathAndBody(t *testing.T) {
	RegisterT(t)
	res := mcpserver.Dispatch(echoEngine{}, origRequest(), commentTool, map[string]any{"number": float64(12), "id": float64(5), "content": "hi"})

	Expect(res.IsError).IsFalse()
	got := decode(t, res.Text)
	Expect(got["method"]).Equals("PUT")
	Expect(got["path"]).Equals("/api/v1/posts/12/comments/5")
	Expect(got["body"]).Equals(`{"content":"hi"}`)
	Expect(got["auth"]).Equals("Bearer tok")
	Expect(got["host"]).Equals("demo.test")
	Expect(got["proto"]).Equals("https")
}

func TestDispatch_QueryForGetWithLimitCap(t *testing.T) {
	RegisterT(t)
	search := mcpserver.Tool{Name: "upvora_ideas_search", Method: "GET", Path: "/api/v1/posts", Paged: true,
		Fields: []mcpserver.Field{{Name: "query", Type: "string", In: "query"}, {Name: "limit", Type: "integer", In: "query"}}}

	res := mcpserver.Dispatch(echoEngine{}, origRequest(), search, map[string]any{"query": "dark mode", "limit": float64(5000)})
	got := decode(t, res.Text)
	Expect(got["path"]).Equals("/api/v1/posts")
	Expect(got["query"]).Equals("limit=100&query=dark+mode")
	Expect(got["body"]).Equals("")
}

// Arguments can never steer the request to another path or host.
func TestDispatch_PathParamsAreEscapedAndValidated(t *testing.T) {
	RegisterT(t)
	tagTool := mcpserver.Tool{Name: "upvora_tags_delete", Method: "DELETE", Path: "/api/v1/tags/{slug}",
		Fields: []mcpserver.Field{{Name: "slug", Type: "string", In: "path", Required: true}}}

	res := mcpserver.Dispatch(echoEngine{}, origRequest(), tagTool, map[string]any{"slug": "../admin/settings?x=1"})
	Expect(res.IsError).IsTrue()

	res = mcpserver.Dispatch(echoEngine{}, origRequest(), tagTool, map[string]any{"slug": "a/b?c#d"})
	got := decode(t, res.Text)
	Expect(got["path"]).Equals("/api/v1/tags/a%2Fb%3Fc%23d")

	res = mcpserver.Dispatch(echoEngine{}, origRequest(), tagTool, map[string]any{})
	Expect(res.IsError).IsTrue()
	Expect(res.Text).ContainsSubstring("slug")
}

func TestDispatch_HTTPErrorIsToolError(t *testing.T) {
	RegisterT(t)
	res := mcpserver.Dispatch(echoEngine{status: http.StatusForbidden}, origRequest(), commentTool,
		map[string]any{"number": float64(1), "id": float64(2), "content": "x"})
	Expect(res.IsError).IsTrue()
	Expect(res.Text).ContainsSubstring("403")
}

type bigEngine struct{}

func (bigEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(strings.Repeat("x", 200*1024)))
}

func TestDispatch_TruncatesLargeOutput(t *testing.T) {
	RegisterT(t)
	res := mcpserver.Dispatch(bigEngine{}, origRequest(), commentTool, map[string]any{"number": float64(1), "id": float64(2), "content": "x"})
	Expect(len(res.Text) < 70*1024).IsTrue()
	Expect(res.Text).ContainsSubstring("truncated")
}

func TestVisible_RoleScopeAndConditions(t *testing.T) {
	RegisterT(t)
	member := &entity.User{Role: enum.RoleVisitor}
	admin := &entity.User{Role: enum.RoleAdministrator}
	tenant := &entity.Tenant{IsPro: true}

	adminTool := mcpserver.Tool{Method: "POST", MinRole: enum.RoleAdministrator}
	readTool := mcpserver.Tool{Method: "GET", MinRole: enum.RoleVisitor}
	writeTool := mcpserver.Tool{Method: "POST", MinRole: enum.RoleVisitor}
	billingTool := mcpserver.Tool{Method: "GET", MinRole: enum.RoleAdministrator, Billing: true}

	Expect(mcpserver.Visible(adminTool, member, tenant, oauthas.ScopeFull)).IsFalse()
	Expect(mcpserver.Visible(adminTool, admin, tenant, oauthas.ScopeFull)).IsTrue()
	Expect(mcpserver.Visible(writeTool, member, tenant, oauthas.ScopeRead)).IsFalse()
	Expect(mcpserver.Visible(readTool, member, tenant, oauthas.ScopeRead)).IsTrue()
	Expect(mcpserver.Visible(billingTool, admin, tenant, oauthas.ScopeFull)).IsFalse() // billing disabled in tests
}

func TestDispatch_RawBodyAndBoth(t *testing.T) {
	RegisterT(t)
	importTool := mcpserver.Tool{Name: "upvora_tags_import", Method: "POST", Path: "/api/v1/admin/import/tags",
		Fields: []mcpserver.Field{{Name: "tags", Type: "array", Items: "object", In: "raw", Required: true}}}
	res := mcpserver.Dispatch(echoEngine{}, origRequest(), importTool, map[string]any{"tags": []any{map[string]any{"name": "ux"}}})
	Expect(decode(t, res.Text)["body"]).Equals(`[{"name":"ux"}]`)

	statusTool := mcpserver.Tool{Name: "upvora_oauth_providers_set_status", Method: "POST", Path: "/api/v1/admin/oauth/{provider}/status",
		Fields: []mcpserver.Field{{Name: "provider", Type: "string", In: "both", Required: true}, {Name: "isEnabled", Type: "boolean", In: "body"}}}
	res = mcpserver.Dispatch(echoEngine{}, origRequest(), statusTool, map[string]any{"provider": "_abc", "isEnabled": true})
	got := decode(t, res.Text)
	Expect(got["path"]).Equals("/api/v1/admin/oauth/_abc/status")
	Expect(got["body"]).Equals(`{"isEnabled":true,"provider":"_abc"}`)
}
