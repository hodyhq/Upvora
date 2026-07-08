-- Trigger-status setting: which status_slug causes a scorecard to auto-create
-- when a post transitions to it. NULL = no auto-trigger.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS scorecard_trigger_status_slug VARCHAR(60) NULL;
