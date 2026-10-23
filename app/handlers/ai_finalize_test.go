package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/oauthas"
)

// Finalize vouches for the brief it produced: a short-lived token bound to
// the person and the brief, which a Vora-only site requires to submit.
func TestAIFinalize_IssuesAVoraToken(t *testing.T) {
	RegisterT(t)
	handlers.SetAISiteLimit(handlers.DefaultAISiteLimit)
	bus.AddHandler(func(ctx context.Context, q *query.GetAIAgentForProduct) error {
		q.Result = &entity.AIAgent{Enabled: true}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetAllTags) error { return nil })
	bus.AddHandler(func(ctx context.Context, c *cmd.AIChatCompletion) error {
		c.Result = "===TITLE===\nA better export\n===DESCRIPTION===\nShort.\n===TAGS===\n\n===BRIEF===\n## Problem\nExports are slow."
		return nil
	})
	tenant := *mock.DemoTenant
	tenant.AIEnabled = true
	code, res := mock.NewServer().OnTenant(&tenant).AsUser(mock.AryaStark).ExecutePostAsJSON(handlers.AIFinalize(),
		`{"productId": 0, "messages": [{"role": "user", "content": "exports are slow"}, {"role": "assistant", "content": "tell me more"}]}`)
	Expect(code).Equals(http.StatusOK)
	brief := res.String("brief")
	claims, err := jwt.DecodeVoraBrief(res.String("voraToken"))
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(mock.AryaStark.ID)
	Expect(claims.TenantID).Equals(tenant.ID)
	Expect(claims.BriefHash).Equals(oauthas.HashToken(brief))
}
