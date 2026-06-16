CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Physical warehouses
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

-- Stock per SKU per warehouse — the hot row all reservation math happens on.
CREATE TABLE IF NOT EXISTS inventory (
    id                  UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    sku_id              UUID    NOT NULL, -- logical FK to skus.id in product-catalog DB
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

-- Every stock movement recorded here (immutable append-only ledger)
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

-- Active reservations tied to an order
CREATE TABLE IF NOT EXISTS inventory_reservations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_id UUID        NOT NULL REFERENCES inventory (id),
    order_id     UUID        NOT NULL,
    sku_id       UUID        NOT NULL,
    qty          INT         NOT NULL CHECK (qty > 0),
    status       VARCHAR(50) NOT NULL DEFAULT 'RESERVED'
                 CHECK (status IN ('RESERVED', 'CONFIRMED', 'RELEASED')),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_reservations_order   ON inventory_reservations (order_id);
CREATE INDEX IF NOT EXISTS idx_reservations_expires ON inventory_reservations (expires_at)
    WHERE status = 'RESERVED';

-- Single default warehouse for this stage — the proto/RPC surface doesn't
-- take a warehouse_id yet (see pkg/proto/inventory/inventory.proto), so
-- every SKU's stock lives here until multi-warehouse routing is built.
INSERT INTO warehouses (id, name, city, state, pincode)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Warehouse', 'Bengaluru', 'Karnataka', '560001')
ON CONFLICT (id) DO NOTHING;
