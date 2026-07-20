package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/getfider/fider/app"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
)

type dbAIAgent struct {
	ID           int         `db:"id"`
	ProductID    dbx.NullInt `db:"product_id"`
	Description  string      `db:"description"`
	Instructions string      `db:"instructions"`
	Enabled      bool        `db:"enabled"`
	WebSearch    bool        `db:"web_search_enabled"`
}

func (a *dbAIAgent) toModel() *entity.AIAgent {
	m := &entity.AIAgent{
		ID:               a.ID,
		Description:      a.Description,
		Instructions:     a.Instructions,
		Enabled:          a.Enabled,
		WebSearchEnabled: a.WebSearch,
	}
	if a.ProductID.Valid {
		id := int(a.ProductID.Int64)
		m.ProductID = &id
	}
	return m
}

func setAISettings(ctx context.Context, c *cmd.SetAISettings) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		// The provider key and the web-search key are each write-only: an empty
		// value means "keep what's stored", so the SET list is built dynamically.
		set := []string{
			"ai_enabled", "ai_provider", "ai_model", "ai_custom_base_url", "ai_custom_model",
			"ai_web_search_enabled", "ai_web_search_provider", "ai_web_search_base_url",
		}
		args := []any{
			c.Enabled, c.Provider, c.Model, c.CustomBaseURL, c.CustomModel,
			c.WebSearchEnabled, c.WebSearchProvider, c.WebSearchBaseURL,
		}
		if c.APIKey != "" {
			set = append(set, "ai_api_key")
			args = append(args, c.APIKey)
		}
		if c.WebSearchAPIKey != "" {
			set = append(set, "ai_web_search_api_key")
			args = append(args, c.WebSearchAPIKey)
		}
		assigns := make([]string, len(set))
		for i, col := range set {
			assigns[i] = fmt.Sprintf("%s = $%d", col, i+1)
		}
		args = append(args, tenant.ID)
		q := fmt.Sprintf("UPDATE tenants SET %s WHERE id = $%d", strings.Join(assigns, ", "), len(args))
		if _, err := trx.Execute(q, args...); err != nil {
			return errors.Wrap(err, "failed to set AI settings")
		}
		tenant.AIEnabled = c.Enabled
		return nil
	})
}

func upsertAIAgent(ctx context.Context, c *cmd.UpsertAIAgent) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		var productID interface{}
		if c.ProductID != nil {
			productID = *c.ProductID
		}
		_, err := trx.Execute(`
			INSERT INTO ai_agents (tenant_id, product_id, description, instructions, enabled, web_search_enabled)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, COALESCE(product_id, 0)) DO UPDATE
			SET description = EXCLUDED.description, instructions = EXCLUDED.instructions, enabled = EXCLUDED.enabled, web_search_enabled = EXCLUDED.web_search_enabled`,
			tenant.ID, productID, c.Description, c.Instructions, c.Enabled, c.WebSearchEnabled)
		if err != nil {
			return errors.Wrap(err, "failed to upsert AI agent")
		}
		return nil
	})
}

func listAIAgents(ctx context.Context, q *query.ListAIAgents) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		q.Result = make([]*entity.AIAgent, 0)
		var agents []*dbAIAgent
		err := trx.Select(&agents, `
			SELECT id, product_id, description, instructions, enabled, web_search_enabled
			FROM ai_agents WHERE tenant_id = $1
			ORDER BY COALESCE(product_id, 0)`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list AI agents")
		}
		for _, a := range agents {
			q.Result = append(q.Result, a.toModel())
		}
		return nil
	})
}

func getAIAgentForProduct(ctx context.Context, q *query.GetAIAgentForProduct) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		q.Result = nil
		agent := &dbAIAgent{}
		// Product-specific agent first, then the tenant default.
		err := trx.Get(agent, `
			SELECT id, product_id, description, instructions, enabled, web_search_enabled
			FROM ai_agents
			WHERE tenant_id = $1 AND (COALESCE(product_id, 0) = $2 OR product_id IS NULL)
			ORDER BY COALESCE(product_id, 0) DESC
			LIMIT 1`, tenant.ID, q.ProductID)
		if err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return nil
			}
			return errors.Wrap(err, "failed to get AI agent")
		}
		m := agent.toModel()
		if !m.Enabled {
			return nil
		}
		q.Result = m
		return nil
	})
}

func saveIdeaBrief(ctx context.Context, c *cmd.SaveIdeaBrief) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		_, err := trx.Execute(`
			INSERT INTO idea_briefs (tenant_id, post_id, content, transcript, submitter_user_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, post_id) DO UPDATE
			SET content = EXCLUDED.content, transcript = EXCLUDED.transcript, submitter_user_id = EXCLUDED.submitter_user_id`,
			tenant.ID, c.PostID, c.Content, c.Transcript, c.SubmitterUserID)
		if err != nil {
			return errors.Wrap(err, "failed to save idea brief")
		}
		return nil
	})
}

type dbIdeaBrief struct {
	PostID          int         `db:"post_id"`
	Content         string      `db:"content"`
	Transcript      string      `db:"transcript"`
	SubmitterUserID dbx.NullInt `db:"submitter_user_id"`
}

func getIdeaBrief(ctx context.Context, q *query.GetIdeaBrief) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		q.Result = nil
		b := &dbIdeaBrief{}
		err := trx.Get(b, `
			SELECT post_id, content, transcript, submitter_user_id
			FROM idea_briefs WHERE tenant_id = $1 AND post_id = $2`, tenant.ID, q.PostID)
		if err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return nil
			}
			return errors.Wrap(err, "failed to get idea brief")
		}
		brief := &entity.IdeaBrief{PostID: b.PostID, Content: b.Content, Transcript: b.Transcript}
		if b.SubmitterUserID.Valid {
			id := int(b.SubmitterUserID.Int64)
			brief.SubmitterUserID = &id
		}
		q.Result = brief
		return nil
	})
}
