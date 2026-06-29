CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE reviews (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID NOT NULL,
    sku_id            UUID NOT NULL,
    user_id           UUID NOT NULL,
    order_id          UUID NOT NULL,
    verified_purchase BOOLEAN NOT NULL DEFAULT FALSE,
    rating            SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title             TEXT NOT NULL,
    body              TEXT,
    image_urls        TEXT[] NOT NULL DEFAULT '{}',
    helpful_count     INT NOT NULL DEFAULT 0,
    status            VARCHAR(32) NOT NULL DEFAULT 'PENDING_MODERATION',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, order_id, sku_id)
);

CREATE TABLE return_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL,
    order_item_id UUID NOT NULL,
    user_id       UUID NOT NULL,
    reason        VARCHAR(64) NOT NULL,
    description   TEXT,
    status        VARCHAR(32) NOT NULL DEFAULT 'REQUESTED',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reviews_product ON reviews(product_id, status);
CREATE INDEX idx_reviews_user ON reviews(user_id);
CREATE INDEX idx_returns_order ON return_requests(order_id);
