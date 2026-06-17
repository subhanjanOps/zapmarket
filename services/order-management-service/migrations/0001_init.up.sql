CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    idempotency_key UUID        NOT NULL UNIQUE,
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING'
                    CHECK (status IN ('PENDING', 'RESERVED', 'PAID', 'CONFIRMED', 'CANCELLED')),
    total_amount    BIGINT      NOT NULL,
    currency        CHAR(3)     NOT NULL DEFAULT 'INR',
    payment_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orders_user   ON orders (user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status)  WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS order_items (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       UUID        NOT NULL REFERENCES orders (id),
    sku_id         UUID        NOT NULL,
    quantity       INT         NOT NULL CHECK (quantity > 0),
    unit_price     BIGINT      NOT NULL CHECK (unit_price > 0),
    reservation_id UUID,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);

-- Transactional outbox for order events. Rows are written in the same
-- transaction as the status transition that triggers them. Debezium/Kafka
-- drains these in Stage 7 — until then published_at stays NULL.
CREATE TABLE IF NOT EXISTS outbox (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id   UUID         NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type     VARCHAR(100) NOT NULL,
    payload        JSONB        NOT NULL,
    published_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
