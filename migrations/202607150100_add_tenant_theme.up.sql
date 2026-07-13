-- Theme as settings, not stylesheets (fork v0.36.1.2.65).
-- theme_primary: hex brand color ('' = the built-in Upvora orange).
-- theme_accents: {"buttons"|"votes"|"links"|"header": "#hex"} - per-function
-- accent overrides; missing key = follow the brand. The renderer emits these
-- as CSS TOKEN overrides, so tenant colors survive every redesign. Custom CSS
-- (Advanced) still loads last and wins, unchanged.

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS theme_primary VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS theme_accents JSONB NOT NULL DEFAULT '{}';
