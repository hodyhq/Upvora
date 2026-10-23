package apiv1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers/apiv1"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/oauthas"
	"github.com/getfider/fider/app/pkg/web"
)

// voraSite mocks idea creation on a site with Vora-only submission, and an
// enabled agent when agent is true. It reports whether an idea was created.
func voraSite(agent bool) (*entity.Tenant, *bool) {
	created := false
	bus.AddHandler(func(ctx context.Context, c *cmd.AddNewPost) error {
		created = true
		c.Result = &entity.Post{ID: 1, Number: 1, Title: c.Title}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetPostBySlug) error { return app.ErrNotFound })
	bus.AddHandler(func(ctx context.Context, c *cmd.SetAttachments) error { return nil })
	bus.AddHandler(func(ctx context.Context, c *cmd.AddVote) error { return nil })
	bus.AddHandler(func(ctx context.Context, c *cmd.UploadImages) error { return nil })
	bus.AddHandler(func(ctx context.Context, c *cmd.SaveIdeaBrief) error { return nil })
	bus.AddHandler(func(ctx context.Context, q *query.GetAIAgentForProduct) error {
		if agent {
			q.Result = &entity.AIAgent{Enabled: true}
		}
		return nil
	})
	tenant := *mock.DemoTenant
	tenant.AIEnabled = true
	tenant.VoraRequired = true
	return &tenant, &created
}

const manualIdea = `{"title": "A manual idea without Vora"}`

func TestCreatePost_VoraRequired_BrowserNeedsABrief(t *testing.T) {
	RegisterT(t)
	tenant, created := voraSite(true)
	code, res := mock.NewServer().OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusBadRequest)
	Expect(res.Body.String()).ContainsSubstring("Vora")
	Expect(*created).IsFalse()

	// A brief alone is not proof Vora wrote it: anyone can type one.
	code, _ = mock.NewServer().OnTenant(tenant).AsUser(mock.AryaStark).
		ExecutePost(apiv1.CreatePost(), `{"title": "A brief typed by hand, not Vora", "briefMarkdown": "## Problem\nx"}`)
	Expect(code).Equals(http.StatusBadRequest)
	Expect(*created).IsFalse()

	brief := "## Problem\nx"
	mine, _ := jwt.EncodeVoraBrief(mock.AryaStark.ID, tenant.ID, oauthas.HashToken(brief), time.Hour)
	other, _ := jwt.EncodeVoraBrief(mock.JonSnow.ID, tenant.ID, oauthas.HashToken(brief), time.Hour)
	otherBrief, _ := jwt.EncodeVoraBrief(mock.AryaStark.ID, tenant.ID, oauthas.HashToken("something else"), time.Hour)
	for _, token := range []string{other, otherBrief, "garbage"} {
		body, _ := json.Marshal(map[string]any{"title": "An idea Vora helped with", "briefMarkdown": brief, "voraToken": token})
		code, _ = mock.NewServer().OnTenant(tenant).AsUser(mock.AryaStark).ExecutePost(apiv1.CreatePost(), string(body))
		Expect(code).Equals(http.StatusBadRequest)
	}
	Expect(*created).IsFalse()

	body, _ := json.Marshal(map[string]any{"title": "An idea Vora helped with", "briefMarkdown": brief, "voraToken": mine})
	code, _ = mock.NewServer().OnTenant(tenant).AsUser(mock.AryaStark).ExecutePost(apiv1.CreatePost(), string(body))
	Expect(code).Equals(http.StatusOK)
	Expect(*created).IsTrue()
}

func TestCreatePost_VoraRequired_MCPAndAPIKeysExempt(t *testing.T) {
	RegisterT(t)
	tenant, created := voraSite(true)
	replay := func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			c.Set(oauthas.ReplayCtxKey{}, true)
			return next(c)
		}
	}
	code, _ := mock.NewServer().Use(replay).OnTenant(tenant).AsUser(mock.AryaStark).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusOK)
	Expect(*created).IsTrue()

	*created = false
	apiKey := func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			c.SetAuthenticatedByAPIKey()
			return next(c)
		}
	}
	code, _ = mock.NewServer().Use(apiKey).OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusOK)
	Expect(*created).IsTrue()
}

func TestCreatePost_VoraRequired_OnlyWhereVoraServesTheProduct(t *testing.T) {
	RegisterT(t)
	tenant, created := voraSite(false)
	code, _ := mock.NewServer().OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusOK)
	Expect(*created).IsTrue()

	tenant, created = voraSite(true)
	tenant.VoraRequired = false
	code, _ = mock.NewServer().OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusOK)
	Expect(*created).IsTrue()
}

// On a site with products the form never offers "no product": choosing none
// is not a way around Vora when Vora serves any product.
func TestCreatePost_VoraRequired_NoProductIsNoWayAround(t *testing.T) {
	RegisterT(t)
	tenant, created := voraSite(false) // no agent for "no product"
	tenant.Products = []*entity.Product{{ID: 3, Name: "Upvora"}}
	bus.AddHandler(func(ctx context.Context, q *query.ListAIAgents) error {
		pid := 3
		q.Result = []*entity.AIAgent{{ProductID: &pid, Enabled: true}}
		return nil
	})
	code, _ := mock.NewServer().OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusBadRequest)
	Expect(*created).IsFalse()

	// Vora switched off for the whole site: never required
	tenant.AIEnabled = false
	code, _ = mock.NewServer().OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusOK)
	tenant.AIEnabled = true

	// a site without products keeps the per-product rule: no agent, plain form
	tenant.Products = nil
	code, _ = mock.NewServer().OnTenant(tenant).AsUser(mock.JonSnow).ExecutePost(apiv1.CreatePost(), manualIdea)
	Expect(code).Equals(http.StatusOK)
}
