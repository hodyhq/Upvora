-- Optional Markdown snippet shown above the description textarea in the
-- Share Your Idea modal. Admin edits it in Site Settings → General.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS share_idea_instructions TEXT NOT NULL DEFAULT '';
