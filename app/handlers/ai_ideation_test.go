package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
)

func ideationTenant() *entity.Tenant {
	t := *mock.DemoTenant
	t.Products = []*entity.Product{{ID: 4, Name: "Kahalia", Slug: "kahalia", IsActive: true}}
	t.AIEnabled = true
	return &t
}

func mockIdeation() {
	bus.AddHandler(func(ctx context.Context, q *query.GetAIAgentForProduct) error {
		if q.ProductID == 4 {
			q.Result = &entity.AIAgent{Instructions: "Ask about the venue first.", Enabled: true}
			return nil
		}
		q.Result = &entity.AIAgent{Instructions: "Default guidance.", Enabled: true}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetAllTags) error {
		q.Result = []*entity.Tag{{Name: "UX", Slug: "ux", IsPublic: true}, {Name: "Secret", Slug: "secret", IsPublic: false}}
		return nil
	})
}

func TestAIIdeationContext_ProductGuidance(t *testing.T) {
	RegisterT(t)
	mockIdeation()

	code, res := mock.NewServer().OnTenant(ideationTenant()).AsUser(mock.AryaStark).
		WithURL("http://demo.test.fider.io/api/v1/ai/ideation-context?product=4").ExecuteAsJSON(handlers.AIIdeationContext())

	Expect(code).Equals(http.StatusOK)
	Expect(res.String("product.name")).Equals("Kahalia")
	Expect(res.String("interviewGuidance")).Equals("Ask about the venue first.")
	Expect(res.Strings("briefSections")).Equals(handlers.BriefSections)
	Expect(res.ArrayFieldStrings("tags", "slug")).Equals([]string{"ux"}) // public tags only
	Expect(res.String("submitWith")).Equals("upvora_ai_submit_brief")
	Expect(strings.Contains(res.String("privacyRule"), "email")).IsTrue()
}

func TestAIIdeationContext_GeneralAndUnknownProduct(t *testing.T) {
	RegisterT(t)
	mockIdeation()

	code, res := mock.NewServer().OnTenant(ideationTenant()).AsUser(mock.AryaStark).
		WithURL("http://demo.test.fider.io/api/v1/ai/ideation-context").ExecuteAsJSON(handlers.AIIdeationContext())
	Expect(code).Equals(http.StatusOK)
	Expect(res.Contains("product.name")).IsFalse()
	Expect(res.String("interviewGuidance")).Equals("Default guidance.")
	Expect(res.ArrayFieldStrings("products", "name")).Equals([]string{"Kahalia"})

	code, _ = mock.NewServer().OnTenant(ideationTenant()).AsUser(mock.AryaStark).
		WithURL("http://demo.test.fider.io/api/v1/ai/ideation-context?product=99").Execute(handlers.AIIdeationContext())
	Expect(code).Equals(http.StatusBadRequest)
}

func TestAIIdeationContext_NoAgentConfigured(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.GetAIAgentForProduct) error { return app.ErrNotFound })
	bus.AddHandler(func(ctx context.Context, q *query.GetAllTags) error { return nil })

	code, res := mock.NewServer().OnTenant(ideationTenant()).AsUser(mock.AryaStark).
		WithURL("http://demo.test.fider.io/api/v1/ai/ideation-context").ExecuteAsJSON(handlers.AIIdeationContext())
	Expect(code).Equals(http.StatusOK)
	Expect(res.String("interviewGuidance")).Equals("")
	Expect(len(res.Strings("interviewRules")) > 3).IsTrue()
}

func TestAIIdeationContext_NoGuidanceWhenAIDisabled(t *testing.T) {
	RegisterT(t)
	mockIdeation()
	tenant := ideationTenant()
	tenant.AIEnabled = false
	code, res := mock.NewServer().OnTenant(tenant).AsUser(mock.AryaStark).
		WithURL("http://demo.test.fider.io/api/v1/ai/ideation-context?product=4").ExecuteAsJSON(handlers.AIIdeationContext())
	Expect(code).Equals(http.StatusOK)
	Expect(res.String("interviewGuidance")).Equals("")
}

func TestAIIdeationContext_StoreErrorsAreNotHidden(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.GetAIAgentForProduct) error { return errors.New("database is down") })
	bus.AddHandler(func(ctx context.Context, q *query.GetAllTags) error { return nil })
	code, _ := mock.NewServer().OnTenant(ideationTenant()).AsUser(mock.AryaStark).
		WithURL("http://demo.test.fider.io/api/v1/ai/ideation-context").Execute(handlers.AIIdeationContext())
	Expect(code).Equals(http.StatusInternalServerError)
}

// One site cannot spend unbounded LLM calls, however many accounts call Vora.
func TestAISiteRateAllow(t *testing.T) {
	RegisterT(t)
	handlers.SetAISiteLimit(2)
	defer handlers.SetAISiteLimit(handlers.DefaultAISiteLimit)
	Expect(handlers.AISiteRateAllow(1)).IsTrue()
	Expect(handlers.AISiteRateAllow(1)).IsTrue()
	Expect(handlers.AISiteRateAllow(1)).IsFalse()
	Expect(handlers.AISiteRateAllow(2)).IsTrue()
}
