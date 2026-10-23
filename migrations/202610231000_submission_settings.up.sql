-- Vora-only submission and file attachments on ideas (v0.38.2.8.0); both off by default.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS vora_required BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS file_attachments_enabled BOOLEAN NOT NULL DEFAULT false;
