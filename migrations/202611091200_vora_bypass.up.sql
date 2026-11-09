-- Let administrators and collaborators skip "Require Vora for new ideas" (v0.39.0.8.3); both off by default.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS vora_bypass_admins BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS vora_bypass_collaborators BOOLEAN NOT NULL DEFAULT false;
