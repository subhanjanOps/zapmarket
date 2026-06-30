CREATE TABLE delivery_agents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL,
    name         TEXT NOT NULL,
    phone        TEXT NOT NULL,
    vehicle_type TEXT NOT NULL DEFAULT 'BIKE',
    zone         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'AVAILABLE',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_agents_zone ON delivery_agents(zone);
CREATE INDEX idx_delivery_agents_status ON delivery_agents(status);

ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS assigned_agent_id UUID REFERENCES delivery_agents(id),
    ADD COLUMN IF NOT EXISTS attempt_count     INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_attempt_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_attempt_at   TIMESTAMPTZ;

CREATE TABLE proof_of_delivery (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id  UUID NOT NULL REFERENCES shipments(id),
    method       TEXT NOT NULL CHECK (method IN ('OTP','SIGNATURE','PHOTO')),
    otp_verified BOOLEAN NOT NULL DEFAULT FALSE,
    photo_url    TEXT,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_by UUID NOT NULL REFERENCES delivery_agents(id)
);

CREATE INDEX idx_pod_shipment ON proof_of_delivery(shipment_id);

CREATE TABLE cod_reconciliations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id  UUID NOT NULL REFERENCES shipments(id),
    agent_id     UUID NOT NULL REFERENCES delivery_agents(id),
    amount_paise BIGINT NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remitted_at  TIMESTAMPTZ,
    status       TEXT NOT NULL DEFAULT 'COLLECTED' CHECK (status IN ('COLLECTED', 'REMITTED'))
);

CREATE INDEX idx_cod_recon_shipment ON cod_reconciliations(shipment_id);

CREATE TABLE outbox (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic        TEXT NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;
