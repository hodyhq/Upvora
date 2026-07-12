package postgres

import (
	"context"
	"fmt"

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
	{"score_strategic", "Strategic alignment", 20, "Does this advance a stated company goal or priority?", 10},
	{"score_business_value", "Business value", 20, "Will this measurably improve revenue, cost, quality, or speed?", 20},
	{"score_ownership", "Ownership clarity", 15, "Is there a clear owner accountable for the outcome?", 30},
	{"score_workflow", "Workflow clarity", 15, "Do we understand how day-to-day work changes once this ships?", 40},
	{"score_data_readiness", "Readiness", 10, "Are the inputs - data, content, integrations - available and reliable?", 50},
	{"score_risk", "Risk manageability", 10, "Can security, privacy, and compliance risks be managed?", 60},
	{"score_adoption", "Adoption likelihood", 5, "Will people actually use this without heavy change management?", 70},
	{"score_supportability", "Supportability", 5, "Can we support and maintain this for the long haul?", 80},
}

// scorecardDefaultFieldSeeds are the non-system starter fields seeded for new
// tenants. They exercise every field type across every body group so a fresh
// scorecard demonstrates the whole form; admins can edit or delete them.
// Kept in sync with migrations/202607120100_universal_scorecard_defaults.up.sql.
type scorecardDefaultFieldSeed struct {
	Key       string
	Label     string
	GroupKey  string
	Type      string
	Choices   string // empty = NULL
	Question  string
	SortOrder int
}

var scorecardDefaultFieldSeeds = []scorecardDefaultFieldSeed{
	{"requesting_team", "Requesting team", "intake", "text", "", "Which team or department is asking for this?", 110},
	{"requested_by", "Requested by", "intake", "user", "", "Who raised or sponsors this request?", 120},
	{"problem_statement", "Problem statement", "context", "note", "", "What problem does this solve, and what happens if we do nothing?", 210},
	{"reference_link", "Reference link", "context", "url", "", "Link to a doc, mockup, or example that explains the idea.", 220},
	{"process_affected", "Process affected", "workflow", "multiline", "", "Which process or workflow would this change, and how is it handled today?", 310},
	{"needed_by", "Needed by", "workflow", "date", "", "When does this need to be in place to matter?", 320},
	{"business_owner", "Business owner", "ownership", "user", "", "Who owns this after it ships?", 410},
	{"effort_estimate", "Effort estimate", "classification", "choice", `[
  {"value": "Small",       "color": "mint"},
  {"value": "Medium",      "color": "gold"},
  {"value": "Large",       "color": "salmon"},
  {"value": "Extra Large", "color": "coral"}
]`, "How big is this likely to be?", 510},
	{"people_impacted", "People impacted", "classification", "number", "", "Roughly how many people would use or benefit from this?", 520},
	{"decision_notes", "Decision notes", "decision", "note", "", "Rationale, conditions, and next steps for the decision.", 910},
	{"next_review", "Next review", "decision", "date", "", "When should this decision be revisited?", 920},
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
		if err == app.ErrNotFound {
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
		// Friendly duplicate check before the INSERT: hitting the UNIQUE
		// (tenant_id, key) constraint would surface as an opaque 500.
		var taken bool
		if err := trx.Scalar(&taken, `SELECT EXISTS(SELECT 1 FROM scorecard_fields WHERE tenant_id = $1 AND key = $2)`, tenant.ID, c.Key); err == nil && taken {
			return fmt.Errorf("a field with key %q already exists — edit that field instead, or pick a different key", c.Key)
		}
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
		// History rule: a question can only be deleted while no card has ever
		// answered it. Empty string and '0' (score "not scored") don't count
		// as answers. Once answered, the field must be deactivated instead so
		// old cards keep rendering it.
		var key string
		err := trx.Scalar(&key, `SELECT key FROM scorecard_fields WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ID)
		if err == app.ErrNotFound {
			return app.ErrNotFound
		}
		if err != nil {
			return errors.Wrap(err, "failed to look up scorecard field %d", c.ID)
		}
		var answered int
		err = trx.Scalar(&answered, `
			SELECT COUNT(*) FROM scorecards
			WHERE tenant_id = $1 AND COALESCE(values->>$2, '') NOT IN ('', '0')
		`, tenant.ID, key)
		if err != nil {
			return errors.Wrap(err, "failed to count answers for scorecard field %q", key)
		}
		if answered > 0 {
			// Plain fmt error: the message surfaces verbatim in the admin UI's
			// 400 response, so no file:line suffix from pkg/errors.New.
			return fmt.Errorf("this question has answers on %d scorecard(s) and cannot be deleted — deactivate it instead; it will stay visible on those cards and stop appearing on new ones", answered)
		}
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

type dbScorecardFieldUsage struct {
	Key   string `db:"key"`
	Count int    `db:"count"`
}

// getScorecardFieldUsage counts, per field key, how many cards hold a real
// answer for it (empty and '0' excluded). Feeds the admin page's delete-vs-
// deactivate gating.
func getScorecardFieldUsage(ctx context.Context, q *query.GetScorecardFieldUsage) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbScorecardFieldUsage{}
		err := trx.Select(&rows, `
			SELECT e.key, COUNT(*) AS count
			FROM scorecards s, LATERAL jsonb_each_text(s.values) AS e(key, value)
			WHERE s.tenant_id = $1 AND COALESCE(e.value, '') NOT IN ('', '0')
			GROUP BY e.key
		`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to aggregate scorecard field usage")
		}
		q.Result = make(map[string]int, len(rows))
		for _, r := range rows {
			q.Result[r.Key] = r.Count
		}
		return nil
	})
}

// scorecardStatusChoicesSeed mirrors the 202607081800 migration seed — keep in sync.
const scorecardStatusChoicesSeed = `[
  {"value": "Submitted",            "color": "blue",   "bucket": "new"},
  {"value": "Under Review",         "color": "gold",   "bucket": "review"},
  {"value": "Needs Clarification",  "color": "salmon", "bucket": "review"},
  {"value": "Scored",               "color": "lavender", "bucket": "executive"},
  {"value": "Approved for Pilot",   "color": "mint",   "bucket": "executive"},
  {"value": "Pilot in Progress",    "color": "cyan",   "bucket": "executive"},
  {"value": "Approved for Rollout", "color": "purple", "bucket": "executive"},
  {"value": "Deferred",             "color": "gray",   "bucket": "executive"},
  {"value": "Rejected",             "color": "coral",  "bucket": "executive"},
  {"value": "Completed",            "color": "pink",   "bucket": "executive"}
]`

func seedTenantScorecardFields(ctx context.Context, c *cmd.SeedTenantScorecardFields) error {
	return using(ctx, func(trx *dbx.Trx, _ *entity.Tenant, _ *entity.User) error {
		_, err := trx.Execute(`
			INSERT INTO scorecard_fields
				(tenant_id, key, label, group_key, type, choices, sort_order, is_system, is_active)
			VALUES ($1, 'status', 'Status', 'header', 'choice', $2::jsonb, 0, TRUE, TRUE)
			ON CONFLICT (tenant_id, key) DO NOTHING
		`, c.TenantID, scorecardStatusChoicesSeed)
		if err != nil {
			return errors.Wrap(err, "failed to seed scorecard status field for tenant %d", c.TenantID)
		}
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
		for _, seed := range scorecardDefaultFieldSeeds {
			var choices interface{}
			if seed.Choices != "" {
				choices = seed.Choices
			}
			_, err := trx.Execute(`
				INSERT INTO scorecard_fields
					(tenant_id, key, label, group_key, type, choices, weight, question, sort_order, is_system, is_active)
				VALUES ($1, $2, $3, $4, $5, $6::jsonb, 0, $7, $8, FALSE, TRUE)
				ON CONFLICT (tenant_id, key) DO NOTHING
			`, c.TenantID, seed.Key, seed.Label, seed.GroupKey, seed.Type, choices, seed.Question, seed.SortOrder)
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
		if err == app.ErrNotFound {
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
		if err == app.ErrNotFound {
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
		// trx.Get returns app.ErrNotFound (not sql.ErrNoRows) on empty result.
		if c.PostID != nil {
			existing := dbEntities.Scorecard{}
			err := trx.Get(&existing, `SELECT `+scorecardSelectCols+` FROM scorecards WHERE tenant_id = $1 AND post_id = $2`, tenant.ID, *c.PostID)
			if err == nil {
				c.Result = existing.ToModel()
				return nil
			}
			if err != app.ErrNotFound {
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
		rows := []*dbEntities.ScorecardListItem{}
		// Post join keeps the dashboard's number/author/votes columns live —
		// they render current post data on every load, never a stored copy.
		err := trx.Select(&rows, `
			SELECT s.id, s.tenant_id, s.post_id, s.title, s.values::text AS values, s.created_at, s.updated_at,
			       p.number AS post_number, p.slug AS post_slug, u.name AS submitted_by,
			       COALESCE((SELECT COUNT(*) FROM post_votes v WHERE v.post_id = p.id AND v.tenant_id = s.tenant_id), 0) AS post_votes
			FROM scorecards s
			LEFT JOIN posts p ON p.id = s.post_id AND p.tenant_id = s.tenant_id
			LEFT JOIN users u ON u.id = p.user_id AND u.tenant_id = s.tenant_id
			WHERE s.tenant_id = $1
			ORDER BY s.updated_at DESC, s.id DESC
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
				scorecard_band_strong_label = $6,
				scorecard_band_good_label = $7,
				scorecard_band_refine_label = $8,
				scorecard_band_low_label = $9,
				scorecard_band_none_label = $10,
				scorecard_trigger_status_slug = $11
			WHERE id = $12
		`, c.IsEnabled, c.BandStrong, c.BandGood, c.BandRefine, c.BandLow,
			c.BandStrongLabel, c.BandGoodLabel, c.BandRefineLabel, c.BandLowLabel, c.BandNoneLabel,
			triggerArg, tenant.ID)
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
