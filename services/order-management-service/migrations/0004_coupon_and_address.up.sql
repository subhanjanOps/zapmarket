-- Coupon support
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS coupon_code   TEXT,
    ADD COLUMN IF NOT EXISTS discount_paise BIGINT NOT NULL DEFAULT 0;

-- Shipping/delivery address (collected at checkout)
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS delivery_full_name   TEXT,
    ADD COLUMN IF NOT EXISTS delivery_phone       TEXT,
    ADD COLUMN IF NOT EXISTS delivery_address_line1 TEXT,
    ADD COLUMN IF NOT EXISTS delivery_city        TEXT,
    ADD COLUMN IF NOT EXISTS delivery_pincode     TEXT,
    ADD COLUMN IF NOT EXISTS delivery_country     TEXT NOT NULL DEFAULT 'IN';
