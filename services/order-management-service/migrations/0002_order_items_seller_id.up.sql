ALTER TABLE order_items ADD COLUMN IF NOT EXISTS seller_id UUID;

CREATE INDEX IF NOT EXISTS idx_order_items_seller ON order_items (seller_id) WHERE seller_id IS NOT NULL;
