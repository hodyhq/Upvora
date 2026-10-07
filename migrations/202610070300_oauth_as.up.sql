CREATE TABLE IF NOT EXISTS oauth_clients (
  id               SERIAL PRIMARY KEY,
  tenant_id        INT NOT NULL REFERENCES tenants(id),
  client_id        VARCHAR(64) NOT NULL,
  name             VARCHAR(200) NOT NULL,
  redirect_uris    TEXT[] NOT NULL,
  created_by_admin BOOLEAN NOT NULL DEFAULT false,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, client_id)
);

CREATE TABLE IF NOT EXISTS oauth_codes (
  code_hash      VARCHAR(64) PRIMARY KEY,
  tenant_id      INT NOT NULL REFERENCES tenants(id),
  client_id      VARCHAR(64) NOT NULL,
  user_id        INT NOT NULL REFERENCES users(id),
  redirect_uri   TEXT NOT NULL,
  scope          VARCHAR(100) NOT NULL,
  code_challenge VARCHAR(128) NOT NULL,
  expires_at     TIMESTAMPTZ NOT NULL,
  used_at        TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
  token_hash  VARCHAR(64) PRIMARY KEY,
  tenant_id   INT NOT NULL REFERENCES tenants(id),
  client_id   VARCHAR(64) NOT NULL,
  user_id     INT NOT NULL REFERENCES users(id),
  scope       VARCHAR(100) NOT NULL,
  family_id   VARCHAR(64) NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  rotated_at  TIMESTAMPTZ NULL,
  revoked_at  TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS oauth_refresh_tokens_family_idx ON oauth_refresh_tokens (tenant_id, family_id);
CREATE INDEX IF NOT EXISTS oauth_refresh_tokens_client_idx ON oauth_refresh_tokens (tenant_id, client_id);
CREATE INDEX IF NOT EXISTS oauth_codes_client_idx ON oauth_codes (tenant_id, client_id);
