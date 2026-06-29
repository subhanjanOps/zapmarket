CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE wishlist_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    product_id UUID NOT NULL,
    sku_id     UUID,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, product_id)
);

CREATE INDEX idx_wishlist_user ON wishlist_items(user_id);
