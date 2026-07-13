-- Multi-product (fork v0.36.1.2.61): one install, many products.
-- Products are a lens, not a sub-tenant: statuses, tags, members, and settings
-- stay tenant-wide. posts.product_id is nullable - NULL means "General"
-- (unassigned), and deleting a product falls its ideas back to General via
-- ON DELETE SET NULL. Tenants without products behave exactly as before.

CREATE TABLE IF NOT EXISTS products (
    id          SERIAL PRIMARY KEY,
    tenant_id   INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(60)  NOT NULL,
    slug        VARCHAR(60)  NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    color       VARCHAR(16)  NOT NULL DEFAULT '',  -- '' = default (tenant brand color)
    sort_order  INTEGER      NOT NULL DEFAULT 0,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug)
);

ALTER TABLE posts ADD COLUMN IF NOT EXISTS product_id INTEGER NULL REFERENCES products(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS posts_tenant_product_idx ON posts (tenant_id, product_id);
