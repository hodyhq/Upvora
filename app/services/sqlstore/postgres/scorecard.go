package postgres

import (
	"context"
	"database/sql"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/services/sqlstore/dbEntities"
)

// scorecardFieldSelectCols matches the column order in migrations/202607071900_add_scorecard.up.sql.
const scorecardFieldSelectCols = `id, tenant_id, key, label, group_key, type, choices, weight, question,
		sort_order, is_system, is_active, created_at, updated_at`

// scorecardScoringSeeds are the 8 locked scoring dimensions seeded per tenant.
// Kept in sync with the INSERT ... UNION ALL block in the migration so new
// tenants land with the exact same rows as tenants that existed at migration.
type scorecardSeedRow struct {
	Key       string
	Label     string
	Weight    int
	Question  string
	SortOrder int
}

var scorecardScoringSeeds = []scorecardSeedRow{
	{"score_strategic", "Strategic alignment", 20, "Does this support a firm priority?", 10},
	{"score_business_value", "Business value", 20, "Does this improve margin, delivery, quality, risk, speed, or decision-making?", 20},
	{"score_ownership", "Ownership clarity", 15, "Is there a real business owner, not just IT interest?", 30},
	{"score_workflow", "Workflow clarity", 15, "Is the current and future workflow understood?", 40},
	{"score_data_readiness", "Data readiness", 10, "Is the needed data available, trusted, and usable?", 50},
	{"score_risk", "Risk manageability", 10, "Can confidentiality, accuracy, security, and review risks be managed?", 60},
	{"score_adoption", "Adoption likelihood", 5, "Will people actually use this?", 70},
	{"score_supportability", "Supportability", 5, "Can this be supported after the pilot?", 80},
}

func listScorecardFieldsForTenant(ctx context.Context, q *query.ListScorecardFieldsForTenant) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbEntities.ScorecardField{}
		err := trx.Select(&rows, `
			SELECT `+scorecardFieldSelectCols+`
			FROM scorecard_fields
			WHERE tenant_id = $1 AND is_active = TRUE
			ORDER BY sort_order, id
		`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list scorecard fields for tenant %d", tenant.ID)
		}
		q.Result = make([]*entity.ScorecardField, len(rows))
		for i, r := range rows {
			q.Result[i] = r.ToModel()
		}
		return nil
	})
}

func listAllScorecardFieldsForTenant(ctx context.Context, q *query.ListAllScorecardFieldsForTenant) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbEntities.ScorecardField{}
		err := trx.Select(&rows, `
			SELECT `+scorecardFieldSelectCols+`
			FROM scorecard_fields
			WHERE tenant_id = $1
			ORDER BY sort_order, id
		`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list all scorecard fields for tenant %d", tenant.ID)
		}
		q.Result = make([]*entity.ScorecardField, len(rows))
		for i, r := range rows {
			q.Result[i] = r.ToModel()
		}
		return nil
	})
}

func getScorecardFieldByID(ctx context.Context, q *query.GetScorecardFieldByID) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbEntities.ScorecardField{}
		err := trx.Get(&row, `
			SELECT `+scorecardFieldSelectCols+`
			FROM scorecard_fields
			WHERE tenant_id = $1 AND id = $2
		`, tenant.ID, q.ID)
		if err == sql.ErrNoRows {
			return app.ErrNotFound
		}
		if err != nil {
			return errors.Wrap(err, "failed to get scorecard field %d", q.ID)
		}
		q.Result = row.ToModel()
		return nil
	})
}

func createScorecardField(ctx context.Context, c *cmd.CreateScorecardField) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		var choicesArg any
		if len(c.Choices) > 0 {
			choicesArg = string(c.Choices)
		} else {
			choicesArg = nil
		}
		var weightArg any
		if c.Weight != nil {
			weightArg = *c.Weight
		} else {
			weightArg = nil
		}
		var questionArg any
		if c.Question != "" {
			questionArg = c.Question
		} else {
			questionArg = nil
		}
		row := dbEntities.ScorecardField{}
		err := trx.Get(&row, `
			INSERT INTO scorecard_fields
				(tenant_id, key, label, group_key, type, choices, weight, question, sort_order, is_system, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, TRUE)
			RETURNING `+scorecardFieldSelectCols+`
		`, tenant.ID, c.Key, c.Label, c.GroupKey, c.Type, choicesArg, weightArg, questionArg, c.SortOrder)
		if err != nil {
			return errors.Wrap(err, "failed to create scorecard field")
		}
		c.Result = row.ToModel()
		return nil
	})
}

func updateScorecardField(ctx context.Context, c *cmd.UpdateScorecardField) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		var choicesArg any
		if len(c.Choices) > 0 {
			choicesArg = string(c.Choices)
		} else {
			choicesArg = nil
		}
		var weightArg any
		if c.Weight != nil {
			weightArg = *c.Weight
		} else {
			weightArg = nil
		}
		var questionArg any
		if c.Question != "" {
			questionArg = c.Question
		} else {
			questionArg = nil
		}
		_, err := trx.Execute(`
			UPDATE scorecard_fields SET
				label = $1, choices = $2, weight = $3, question = $4,
				sort_order = $5, is_active = $6, updated_at = NOW()
			WHERE tenant_id = $7 AND id = $8
		`, c.Label, choicesArg, weightArg, questionArg, c.SortOrder, c.IsActive, tenant.ID, c.ID)
		if err != nil {
			return errors.Wrap(err, "failed to update scorecard field %d", c.ID)
		}
		return nil
	})
}

func deleteScorecardField(ctx context.Context, c *cmd.DeleteScorecardField) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		res, err := trx.Execute(`DELETE FROM scorecard_fields WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ID)
		if err != nil {
			return errors.Wrap(err, "failed to delete scorecard field %d", c.ID)
		}
		if res == 0 {
			return app.ErrNotFound
		}
		return nil
	})
}

func seedTenantScorecardFields(ctx context.Context, c *cmd.SeedTenantScorecardFields) error {
	return using(ctx, func(trx *dbx.Trx, _ *entity.Tenant, _ *entity.User) error {
		for _, seed := range scorecardScoringSeeds {
			_, err := trx.Execute(`
				INSERT INTO scorecard_fields
					(tenant_id, key, label, group_key, type, weight, question, sort_order, is_system, is_active)
				VALUES ($1, $2, $3, 'scoring', 'score', $4, $5, $6, TRUE, TRUE)
				ON CONFLICT (tenant_id, key) DO NOTHING
			`, c.TenantID, seed.Key, seed.Label, seed.Weight, seed.Question, seed.SortOrder)
			if err != nil {
				return errors.Wrap(err, "failed to seed scorecard field %q for tenant %d", seed.Key, c.TenantID)
			}
		}
		return nil
	})
}

const scorecardSelectCols = `id, tenant_id, post_id, title, values::text AS values, created_at, updated_at`

func getScorecardByID(ctx context.Context, q *query.GetScorecardByID) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbEntities.Scorecard{}
		err := trx.Get(&row, `SELECT `+scorecardSelectCols+` FROM scorecards WHERE tenant_id = $1 AND id = $2`, tenant.ID, q.ID)
		if err == sql.ErrNoRows {
			return app.ErrNotFound
		}
		if err != nil {
			return errors.Wrap(err, "failed to get scorecard by id %d", q.ID)
		}
		q.Result = row.ToModel()
		return nil
	})
}

func getScorecardByPostID(ctx context.Context, q *query.GetScorecardByPostID) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbEntities.Scorecard{}
		err := trx.Get(&row, `SELECT `+scorecardSelectCols+` FROM scorecards WHERE tenant_id = $1 AND post_id = $2`, tenant.ID, q.PostID)
		if err == sql.ErrNoRows {
			return app.ErrNotFound
		}
		if err != nil {
			return errors.Wrap(err, "failed to get scorecard for post %d", q.PostID)
		}
		q.Result = row.ToModel()
		return nil
	})
}

func createScorecard(ctx context.Context, c *cmd.CreateScorecard) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		// Idempotent: if a linked card already exists, return it instead of
		// inserting a duplicate (the UNIQUE index would refuse it anyway).
		if c.PostID != nil {
			existing := dbEntities.Scorecard{}
			err := trx.Get(&existing, `SELECT `+scorecardSelectCols+` FROM scorecards WHERE tenant_id = $1 AND post_id = $2`, tenant.ID, *c.PostID)
			if err == nil {
				c.Result = existing.ToModel()
				return nil
			}
			if err != sql.ErrNoRows {
				return errors.Wrap(err, "failed to check for existing scorecard on post %d", *c.PostID)
			}
		}
		var postIDArg any
		if c.PostID != nil {
			postIDArg = *c.PostID
		} else {
			postIDArg = nil
		}
		title := c.Title
		if title == "" {
			title = "Untitled scorecard"
		}
		var createdByArg any
		if user != nil {
			createdByArg = user.ID
		} else {
			createdByArg = nil
		}
		row := dbEntities.Scorecard{}
		err := trx.Get(&row, `
			INSERT INTO scorecards (tenant_id, post_id, title, values, created_by)
			VALUES ($1, $2, $3, '{}'::JSONB, $4)
			RETURNING `+scorecardSelectCols+`
		`, tenant.ID, postIDArg, title, createdByArg)
		if err != nil {
			return errors.Wrap(err, "failed to create scorecard")
		}
		c.Result = row.ToModel()
		return nil
	})
}

func updateScorecardValues(ctx context.Context, c *cmd.UpdateScorecardValues) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		values := c.Values
		if len(values) == 0 {
			values = []byte("{}")
		}
		res, err := trx.Execute(`
			UPDATE scorecards SET title = $1, values = $2::JSONB, updated_at = NOW()
			WHERE tenant_id = $3 AND id = $4
		`, c.Title, string(values), tenant.ID, c.ID)
		if err != nil {
			return errors.Wrap(err, "failed to update scorecard %d", c.ID)
		}
		if res == 0 {
			return app.ErrNotFound
		}
		return nil
	})
}

func deleteScorecard(ctx context.Context, c *cmd.DeleteScorecard) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		res, err := trx.Execute(`DELETE FROM scorecards WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ID)
		if err != nil {
			return errors.Wrap(err, "failed to delete scorecard %d", c.ID)
		}
		if res == 0 {
			return app.ErrNotFound
		}
		return nil
	})
}

func listScorecardsForTenant(ctx context.Context, q *query.ListScorecardsForTenant) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbEntities.Scorecard{}
		err := trx.Select(&rows, `
			SELECT `+scorecardSelectCols+`
			FROM scorecards
			WHERE tenant_id = $1
			ORDER BY updated_at DESC, id DESC
		`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list scorecards for tenant %d", tenant.ID)
		}
		q.Result = make([]*entity.Scorecard, len(rows))
		for i, r := range rows {
			q.Result[i] = r.ToModel()
		}
		return nil
	})
}

func setTenantScorecardSettings(ctx context.Context, c *cmd.SetTenantScorecardSettings) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		var triggerArg any
		if c.TriggerStatusSlug != "" {
			triggerArg = c.TriggerStatusSlug
		} else {
			triggerArg = nil
		}
		_, err := trx.Execute(`
			UPDATE tenants SET
				is_scorecard_enabled = $1,
				scorecard_band_strong = $2,
				scorecard_band_good = $3,
				scorecard_band_refine = $4,
				scorecard_band_low = $5,
				scorecard_trigger_status_slug = $6
			WHERE id = $7
		`, c.IsEnabled, c.BandStrong, c.BandGood, c.BandRefine, c.BandLow, triggerArg, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to update scorecard settings for tenant %d", tenant.ID)
		}
		tenant.IsScorecardEnabled = c.IsEnabled
		tenant.ScorecardBandStrong = c.BandStrong
		tenant.ScorecardBandGood = c.BandGood
		tenant.ScorecardBandRefine = c.BandRefine
		tenant.ScorecardBandLow = c.BandLow
		tenant.ScorecardTriggerStatusSlug = c.TriggerStatusSlug
		return nil
	})
}
