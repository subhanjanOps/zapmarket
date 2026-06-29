ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS saga_status VARCHAR(32) NOT NULL DEFAULT 'AWAITING_INVENTORY';

CREATE INDEX IF NOT EXISTS idx_orders_saga_status ON orders(saga_status) WHERE saga_status != 'COMPLETE';
