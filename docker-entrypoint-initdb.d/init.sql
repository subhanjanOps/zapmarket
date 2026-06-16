CREATE DATABASE userauth;
CREATE DATABASE ordermgmt;
CREATE DATABASE inventory;
CREATE DATABASE payment;
CREATE DATABASE notification;
CREATE DATABASE productcatalog;

-- userauth and productcatalog schemas are now owned by golang-migrate:
-- see services/auth-service/migrations and
-- services/product-catalog-service/migrations. Only database creation
-- (which migrate cannot do) stays here for those two.
\connect userauth
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect productcatalog
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- inventory schema is now owned by golang-migrate:
-- see services/inventory-service/migrations.
\connect inventory
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- payment schema is now owned by golang-migrate:
-- see services/payment-service/migrations.
\connect payment
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ordermgmt is not yet owned by a service migration directory since that
-- service is still a scaffold. It'll move to services/order-management-service/migrations
-- when that service is built (see planning/05-order-management-saga.md).
\connect ordermgmt
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL,
    status              VARCHAR(50) NOT NULL DEFAULT 'pending'
                        CHECK (status IN (
                            'pending',
                            'confirmed',
                            'processing',
                            'shipped',
                            'delivered',
                            'cancelled',
                            'refunded'
                        )),
    subtotal_amount     BIGINT      NOT NULL,
    discount_amount     BIGINT      NOT NULL DEFAULT 0,
    shipping_amount     BIGINT      NOT NULL DEFAULT 0,
    tax_amount          BIGINT      NOT NULL DEFAULT 0,
    total_amount        BIGINT      NOT NULL,
    currency            CHAR(3)     NOT NULL DEFAULT 'INR',
    idempotency_key     UUID        NOT NULL UNIQUE,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orders_user   ON orders (user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status)  WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS order_shipping_addresses (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID         NOT NULL UNIQUE REFERENCES orders (id),
    line1      TEXT         NOT NULL,
    line2      TEXT,
    city       VARCHAR(100) NOT NULL,
    state      VARCHAR(100) NOT NULL,
    country    CHAR(2)      NOT NULL DEFAULT 'IN',
    pincode    VARCHAR(20)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS order_items (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID         NOT NULL REFERENCES orders (id),
    sku_id        UUID         NOT NULL,
    sku_code      VARCHAR(100) NOT NULL,
    product_name  VARCHAR(500) NOT NULL,
    variant_attrs JSONB        NOT NULL DEFAULT '{}',
    qty           INT          NOT NULL CHECK (qty > 0),
    unit_price    BIGINT       NOT NULL,
    discount      BIGINT       NOT NULL DEFAULT 0,
    total_price   BIGINT       NOT NULL,
    currency      CHAR(3)      NOT NULL DEFAULT 'INR',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_sku   ON order_items (sku_id);

CREATE TABLE IF NOT EXISTS order_status_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID        NOT NULL REFERENCES orders (id),
    from_status VARCHAR(50),
    to_status   VARCHAR(50) NOT NULL,
    reason      TEXT,
    changed_by  UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_status_history_order ON order_status_history (order_id);

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

\connect notification
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Notification service is stateless; no primary tables are created here.
