DROP MATERIALIZED VIEW IF EXISTS product_ratings;
DROP TABLE IF EXISTS return_items;
ALTER TABLE return_requests
    DROP COLUMN IF EXISTS image_urls,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS rejected_at,
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS reverse_shipment_id;
