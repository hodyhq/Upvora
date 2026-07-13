-- Internal collaboration layer (fork v0.36.1.2.59):
-- 1) comments gain is_internal - visible only to collaborators/admins,
--    excluded from public comment counts, never shipped to visitors.
-- 2) post_internal_notes - ONE shared team note per post, editable from the
--    post page and the linked scorecard (same row = always in sync).

ALTER TABLE comments ADD COLUMN IF NOT EXISTS is_internal BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS post_internal_notes (
    tenant_id  INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    content    TEXT NOT NULL DEFAULT '',
    updated_by INTEGER NULL REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, post_id)
);
