-- The refresh family an authorization code started, so reusing the code
-- revokes what it already issued (OAuth 2.1).
ALTER TABLE oauth_codes ADD COLUMN IF NOT EXISTS family_id VARCHAR(64) NULL;
