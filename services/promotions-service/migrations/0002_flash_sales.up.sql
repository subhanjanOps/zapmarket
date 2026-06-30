ALTER TABLE coupons
    ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS ends_at   TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_coupons_active_sale ON coupons (starts_at, ends_at)
    WHERE discount_type = 'PERCENT' AND is_active = TRUE;
