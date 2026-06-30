-- Extend return_requests with image evidence and expanded status
ALTER TABLE return_requests
    ADD COLUMN IF NOT EXISTS image_urls TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT,
    ADD COLUMN IF NOT EXISTS reverse_shipment_id UUID;

-- Drop and recreate status check to include new states
ALTER TABLE return_requests DROP CONSTRAINT IF EXISTS return_requests_status_check;
ALTER TABLE return_requests ADD CONSTRAINT return_requests_status_check
    CHECK (status IN ('REQUESTED', 'APPROVED', 'REJECTED', 'PICKUP_SCHEDULED', 'COMPLETED', 'REFUNDED'));

-- Multi-item return support
CREATE TABLE IF NOT EXISTS return_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    return_request_id UUID NOT NULL REFERENCES return_requests(id),
    order_item_id     UUID NOT NULL,
    quantity          INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    reason            TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_return_items_request ON return_items(return_request_id);

-- Rating aggregation materialized view
CREATE MATERIALIZED VIEW IF NOT EXISTS product_ratings AS
SELECT
    product_id,
    ROUND(AVG(rating)::NUMERIC, 2) AS avg_rating,
    COUNT(*)                        AS review_count
FROM reviews
WHERE status = 'PUBLISHED'
GROUP BY product_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_ratings_product ON product_ratings(product_id);
