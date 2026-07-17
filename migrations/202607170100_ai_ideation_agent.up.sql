-- Vora — the ideation agent (admin-toggleable, per-product instructions).

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_enabled         BOOLEAN      NOT NULL DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_provider        VARCHAR(20)  NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_api_key         TEXT         NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_model           VARCHAR(60)  NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_custom_base_url TEXT         NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_custom_model    VARCHAR(100) NOT NULL DEFAULT '';

-- One agent per product plus a default (product_id NULL) used for General.
-- The agent is always named Vora — no name column by design.
CREATE TABLE IF NOT EXISTS ai_agents (
    id           SERIAL PRIMARY KEY,
    tenant_id    INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id   INTEGER NULL REFERENCES products(id) ON DELETE CASCADE,
    description  TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE UNIQUE INDEX IF NOT EXISTS ai_agents_tenant_product ON ai_agents (tenant_id, COALESCE(product_id, 0));

-- The brief Vora writes with the user. The submitter's email is stored ONLY
-- as the literal token {{submitter_email}} inside content — substitution with
-- the real address happens server-side at admin download time.
CREATE TABLE IF NOT EXISTS idea_briefs (
    tenant_id         INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id           INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    content           TEXT NOT NULL DEFAULT '',
    submitter_user_id INTEGER NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, post_id)
);
