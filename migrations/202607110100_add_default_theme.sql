-- Per-tenant default appearance: light | dark | system.
-- User's own toggle (localStorage) always wins; this only sets the first-visit face.
ALTER TABLE tenants ADD COLUMN default_theme VARCHAR(10) NOT NULL DEFAULT 'light';
