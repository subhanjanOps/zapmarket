CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE shipments (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id             UUID NOT NULL UNIQUE,
    carrier_shipment_id  VARCHAR(128),
    carrier              VARCHAR(64),
    tracking_url         TEXT,
    status               VARCHAR(32) NOT NULL DEFAULT 'CREATED',
    label_url            TEXT,
    estimated_delivery   TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tracking_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id  UUID NOT NULL REFERENCES shipments(id),
    status       VARCHAR(64) NOT NULL,
    description  TEXT,
    location     TEXT,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shipments_order ON shipments(order_id);
CREATE INDEX idx_tracking_shipment ON tracking_events(shipment_id);
