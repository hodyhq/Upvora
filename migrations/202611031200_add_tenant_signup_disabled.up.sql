-- Fider v0.39.0 ships this as 202610131200; renumbered after the fork's 202610231000 so databases already past it still get the column.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS is_signup_disabled BOOLEAN NOT NULL DEFAULT false;
