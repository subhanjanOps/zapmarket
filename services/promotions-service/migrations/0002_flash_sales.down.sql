DROP INDEX IF EXISTS idx_coupons_active_sale;
ALTER TABLE coupons
    DROP COLUMN IF EXISTS starts_at,
    DROP COLUMN IF EXISTS ends_at;
