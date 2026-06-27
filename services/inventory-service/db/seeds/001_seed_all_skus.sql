-- Inserts inventory rows for all SKUs that don't yet have one.
-- Safe to run repeatedly (ON CONFLICT DO NOTHING).
-- Run against the inventory DB:
--   psql -U zapuser -d inventory -f 001_seed_all_skus.sql

INSERT INTO inventory (sku_id, warehouse_id, qty_on_hand, qty_reserved, low_stock_threshold)
SELECT
    s.id::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    100,
    0,
    10
FROM dblink(
    'dbname=productcatalog user=zapuser password=zappass123 host=localhost',
    'SELECT id FROM skus WHERE deleted_at IS NULL'
) AS s(id text)
ON CONFLICT (sku_id, warehouse_id) DO NOTHING;
