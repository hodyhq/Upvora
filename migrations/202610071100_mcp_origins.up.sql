-- MCP-only public addresses (MCP_ORIGINS): codes and refresh tokens issued
-- there remember that address and work only there. '' is the board's own
-- (and every grant issued before this), accepted only off MCP-only addresses.
ALTER TABLE oauth_codes ADD COLUMN IF NOT EXISTS origin TEXT NOT NULL DEFAULT '';
ALTER TABLE oauth_refresh_tokens ADD COLUMN IF NOT EXISTS origin TEXT NOT NULL DEFAULT '';

-- Single-use codes that carry a person signed in on the board's own address
-- back to the MCP-only address to finish connecting.
CREATE TABLE IF NOT EXISTS oauth_signin_handoffs (
  code_hash      VARCHAR(64) PRIMARY KEY,
  tenant_id      INT NOT NULL REFERENCES tenants(id),
  user_id        INT NOT NULL REFERENCES users(id),
  origin         TEXT NOT NULL,
  query          TEXT NOT NULL,
  security_stamp VARCHAR(100) NOT NULL DEFAULT '',
  flow_hash      VARCHAR(64) NOT NULL,
  expires_at     TIMESTAMPTZ NOT NULL,
  used_at        TIMESTAMPTZ NULL
);
