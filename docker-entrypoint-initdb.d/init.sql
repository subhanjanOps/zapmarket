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

-- The schemas below (inventory, ordermgmt, payment) are not yet owned by a
-- service migration directory since those services are still scaffolds.
-- They'll move to services/<name>/migrations when each service is built
-- (see planning/03-inventory-service.md, 04-payment-service.md,
-- 05-order-management-saga.md).
\connect inventory
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS warehouses (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    city       VARCHAR(100) NOT NULL,
    state      VARCHAR(100) NOT NULL,
    pincode    VARCHAR(20)  NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS inventory (
    id                  UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    sku_id              UUID    NOT NULL,
    warehouse_id        UUID    NOT NULL REFERENCES warehouses (id),
    qty_on_hand         INT     NOT NULL DEFAULT 0 CHECK (qty_on_hand >= 0),
    qty_reserved        INT     NOT NULL DEFAULT 0 CHECK (qty_reserved >= 0),
    qty_available       INT     GENERATED ALWAYS AS (qty_on_hand - qty_reserved) STORED,
    low_stock_threshold INT     NOT NULL DEFAULT 10,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    UNIQUE (sku_id, warehouse_id)
);

CREATE INDEX IF NOT EXISTS idx_inventory_sku       ON inventory (sku_id)       WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_warehouse ON inventory (warehouse_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS inventory_ledger (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_id  UUID        NOT NULL REFERENCES inventory (id),
    order_id      UUID,
    movement_type VARCHAR(50) NOT NULL
                  CHECK (movement_type IN (
                      'purchase_order',
                      'reservation',
                      'reservation_release',
                      'sale',
                      'return',
                      'adjustment'
                  )),
    qty_delta     INT         NOT NULL,
    qty_before    INT         NOT NULL,
    qty_after     INT         NOT NULL,
    note          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_inventory ON inventory_ledger (inventory_id);
CREATE INDEX IF NOT EXISTS idx_ledger_order     ON inventory_ledger (order_id) WHERE order_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS inventory_reservations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_id UUID        NOT NULL REFERENCES inventory (id),
    order_id     UUID        NOT NULL,
    sku_id       UUID        NOT NULL,
    qty          INT         NOT NULL CHECK (qty > 0),
    status       VARCHAR(50) NOT NULL DEFAULT 'reserved'
                 CHECK (status IN ('reserved', 'confirmed', 'released')),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_reservations_order   ON inventory_reservations (order_id);
CREATE INDEX IF NOT EXISTS idx_reservations_expires ON inventory_reservations (expires_at)
    WHERE status = 'reserved';

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

\connect payment
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS payments (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         UUID         NOT NULL,
    user_id          UUID         NOT NULL,
    idempotency_key  UUID         NOT NULL UNIQUE,
    status           VARCHAR(50)  NOT NULL DEFAULT 'pending'
                     CHECK (status IN (
                         'pending',
                         'authorised',
                         'captured',
                         'failed',
                         'refunded',
                         'partially_refunded'
                     )),
    amount           BIGINT       NOT NULL,
    currency         CHAR(3)      NOT NULL DEFAULT 'INR',
    gateway          VARCHAR(50)  NOT NULL,
    gateway_txn_id   VARCHAR(255) UNIQUE,
    gateway_response JSONB,
    failure_reason   TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payments_order  ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS payment_taxes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id  UUID        NOT NULL REFERENCES payments (id),
    tax_type    VARCHAR(50) NOT NULL CHECK (tax_type IN ('CGST', 'SGST', 'IGST', 'CESS')),
    rate_bps    INT         NOT NULL,
    base_amount BIGINT      NOT NULL,
    tax_amount  BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID         NOT NULL REFERENCES payments (id),
    entry_type VARCHAR(50)  NOT NULL CHECK (entry_type IN ('debit', 'credit')),
    account    VARCHAR(100) NOT NULL,
    amount     BIGINT       NOT NULL CHECK (amount > 0),
    currency   CHAR(3)      NOT NULL DEFAULT 'INR',
    description TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_payment ON ledger_entries (payment_id);
CREATE INDEX IF NOT EXISTS idx_ledger_account ON ledger_entries (account);

CREATE TABLE IF NOT EXISTS refunds (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id        UUID        NOT NULL REFERENCES payments (id),
    order_id          UUID        NOT NULL,
    amount            BIGINT      NOT NULL,
    currency          CHAR(3)     NOT NULL DEFAULT 'INR',
    reason            VARCHAR(255),
    status            VARCHAR(50) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'processed', 'failed')),
    gateway_refund_id VARCHAR(255),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refunds_payment ON refunds (payment_id);
CREATE INDEX IF NOT EXISTS idx_refunds_order   ON refunds (order_id);

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
