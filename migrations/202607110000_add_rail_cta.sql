-- Add editable sidebar "call to action" fields to tenants (Upvora board rail).
ALTER TABLE tenants ADD COLUMN rail_cta_heading TEXT NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN rail_cta_text TEXT NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN rail_cta_button TEXT NOT NULL DEFAULT '';
