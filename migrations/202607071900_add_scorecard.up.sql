-- Fider Scorecard feature (fork-major 1). Adds per-tenant feature toggle,
-- band thresholds, field catalogue, and cards table. Seeds the 8 locked
-- scoring dimensions for every existing tenant so the weighted-score math
-- has something to compute against out of the gate.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS is_scorecard_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS scorecard_band_strong INTEGER NOT NULL DEFAULT 80,
    ADD COLUMN IF NOT EXISTS scorecard_band_good   INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS scorecard_band_refine INTEGER NOT NULL DEFAULT 40,
    ADD COLUMN IF NOT EXISTS scorecard_band_low    INTEGER NOT NULL DEFAULT 20;

CREATE TABLE IF NOT EXISTS scorecard_fields (
    id         SERIAL PRIMARY KEY,
    tenant_id  INTEGER     NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key        VARCHAR(60) NOT NULL,
    label      VARCHAR(120) NOT NULL,
    group_key  VARCHAR(40) NOT NULL,
    type       VARCHAR(20) NOT NULL,
    choices    JSONB       NULL,
    weight     INTEGER     NULL,
    question   TEXT        NULL,
    sort_order INTEGER     NOT NULL DEFAULT 0,
    is_system  BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, key),
    CHECK (type IN ('text', 'note', 'date', 'number', 'url', 'choice', 'score')),
    CHECK (group_key IN ('intake', 'context', 'workflow', 'ownership', 'classification', 'scoring', 'decision'))
);

CREATE INDEX IF NOT EXISTS scorecard_fields_tenant_active_idx ON scorecard_fields (tenant_id, is_active);

CREATE TABLE IF NOT EXISTS scorecards (
    id         SERIAL PRIMARY KEY,
    tenant_id  INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id    INTEGER NULL REFERENCES posts(id) ON DELETE SET NULL,
    title      TEXT NOT NULL,
    values     JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_by INTEGER NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS scorecards_tenant_post_unique ON scorecards (tenant_id, post_id) WHERE post_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS scorecards_tenant_idx ON scorecards (tenant_id);

-- Seed the 8 locked scoring dimensions for every existing tenant. is_system=TRUE
-- protects them from delete; admins can still edit weight, question, label.
-- Weights sum to 100; math is ROUND(sum(value * weight) / 5).
INSERT INTO scorecard_fields (tenant_id, key, label, group_key, type, weight, question, sort_order, is_system)
SELECT id, 'score_strategic',        'Strategic alignment', 'scoring', 'score', 20, 'Does this support a firm priority?',                                              10, TRUE FROM tenants
UNION ALL
SELECT id, 'score_business_value',   'Business value',      'scoring', 'score', 20, 'Does this improve margin, delivery, quality, risk, speed, or decision-making?', 20, TRUE FROM tenants
UNION ALL
SELECT id, 'score_ownership',        'Ownership clarity',   'scoring', 'score', 15, 'Is there a real business owner, not just IT interest?',                          30, TRUE FROM tenants
UNION ALL
SELECT id, 'score_workflow',         'Workflow clarity',    'scoring', 'score', 15, 'Is the current and future workflow understood?',                                 40, TRUE FROM tenants
UNION ALL
SELECT id, 'score_data_readiness',   'Data readiness',      'scoring', 'score', 10, 'Is the needed data available, trusted, and usable?',                             50, TRUE FROM tenants
UNION ALL
SELECT id, 'score_risk',             'Risk manageability',  'scoring', 'score', 10, 'Can confidentiality, accuracy, security, and review risks be managed?',          60, TRUE FROM tenants
UNION ALL
SELECT id, 'score_adoption',         'Adoption likelihood', 'scoring', 'score', 5,  'Will people actually use this?',                                                  70, TRUE FROM tenants
UNION ALL
SELECT id, 'score_supportability',   'Supportability',      'scoring', 'score', 5,  'Can this be supported after the pilot?',                                          80, TRUE FROM tenants
ON CONFLICT (tenant_id, key) DO NOTHING;
