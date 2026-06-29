CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE cart_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    sku_id        UUID NOT NULL,
    product_id    UUID NOT NULL,
    product_name  TEXT NOT NULL,
    variant_attrs JSONB NOT NULL DEFAULT '{}',
    quantity      INT NOT NULL CHECK (quantity > 0),
    price_at_add  BIGINT NOT NULL,
    currency      VARCHAR(3) NOT NULL DEFAULT 'INR',
    image_url     TEXT,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, sku_id)
);

CREATE INDEX idx_cart_items_user ON cart_items(user_id);
