-- Codes and refresh tokens are bound to the user's security stamp at issue:
-- a block, role change or sign-out everywhere rotates the stamp and so
-- revokes them, like session and access tokens.
ALTER TABLE oauth_codes ADD COLUMN IF NOT EXISTS security_stamp VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE oauth_refresh_tokens ADD COLUMN IF NOT EXISTS security_stamp VARCHAR(100) NOT NULL DEFAULT '';
