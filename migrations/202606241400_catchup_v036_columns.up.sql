-- Catch-up migration: replays the two v0.36.0 upstream migrations whose
-- filenames sort BEFORE our hcm-beta migrations and therefore got skipped
-- on installs that ran ours first.
--
-- The two upstream files are:
--   202606101200_add_tenant_deletion_schedule.sql
--   202606121200_add_tenant_description_template.sql
--
-- Both are pure ADD COLUMN. The IF NOT EXISTS guards make this safe on
-- clean installs where upstream migrations ran first (the columns will
-- already exist and the statements are no-ops) and on hcm-beta installs
-- where they were skipped (we add them now).

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS scheduled_deletion_at TIMESTAMPTZ NULL;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS deletion_requested_by INT NULL;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS deletion_cancel_key   VARCHAR(64) NULL;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS description_template  TEXT NOT NULL DEFAULT '';
