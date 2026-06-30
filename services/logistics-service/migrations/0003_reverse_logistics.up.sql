ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS shipment_type TEXT NOT NULL DEFAULT 'FORWARD' CHECK (shipment_type IN ('FORWARD', 'REVERSE')),
    ADD COLUMN IF NOT EXISTS parent_shipment_id UUID REFERENCES shipments(id);

INSERT INTO outbox (id, topic, payload, created_at) SELECT gen_random_uuid(), 'noop', '{}', NOW() WHERE FALSE;
