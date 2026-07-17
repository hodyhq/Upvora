package postgres

import (
	"context"

	"github.com/getfider/fider/app"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
)

type dbAIAgent struct {
	ID           int             `db:"id"`
	ProductID    dbx.NullInt     `db:"product_id"`
	Description  string          `db:"description"`
	Instructions string          `db:"instructions"`
	Enabled      bool            `db:"enabled"`
}

func (a *dbAIAgent) toModel() *entity.AIAgent {
	m := &entity.AIAgent{
		ID:           a.ID,
		Description:  a.Description,
		Instructions: a.Instructions,
		Enabled:      a.Enabled,
	}
	if a.ProductID.Valid {
		id := int(a.ProductID.Int64)
		m.ProductID = &id
	}
	return m
}

func setAISettings(ctx context.Context, c *cmd.SetAISettings) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		// An empty key keeps the stored one — the key is write-only client-side.
		if c.APIKey != "" {
			_, err := trx.Execute(`
				UPDATE tenants SET ai_enabled = $1, ai_provider = $2, ai_api_key = $3, ai_model = $4, ai_custom_base_url = $5, ai_custom_model = $6
				WHERE id = $7`,
				c.Enabled, c.Provider, c.APIKey, c.Model, c.CustomBaseURL, c.CustomModel, tenant.ID)
			if err != nil {
				return errors.Wrap(err, "failed to set AI settings")
			}
		} else {
			_, err := trx.Execute(`
				UPDATE tenants SET ai_enabled = $1, ai_provider = $2, ai_model = $3, ai_custom_base_url = $4, ai_custom_model = $5
				WHERE id = $6`,
				c.Enabled, c.Provider, c.Model, c.CustomBaseURL, c.CustomModel, tenant.ID)
			if err != nil {
				return errors.Wrap(err, "failed to set AI settings")
			}
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
			INSERT INTO ai_agents (tenant_id, product_id, description, instructions, enabled)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, COALESCE(product_id, 0)) DO UPDATE
			SET description = EXCLUDED.description, instructions = EXCLUDED.instructions, enabled = EXCLUDED.enabled`,
			tenant.ID, productID, c.Description, c.Instructions, c.Enabled)
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
			SELECT id, product_id, description, instructions, enabled
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
			SELECT id, product_id, description, instructions, enabled
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
