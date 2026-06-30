ALTER TABLE orders
    DROP COLUMN IF EXISTS coupon_code,
    DROP COLUMN IF EXISTS discount_paise,
    DROP COLUMN IF EXISTS delivery_full_name,
    DROP COLUMN IF EXISTS delivery_phone,
    DROP COLUMN IF EXISTS delivery_address_line1,
    DROP COLUMN IF EXISTS delivery_city,
    DROP COLUMN IF EXISTS delivery_pincode,
    DROP COLUMN IF EXISTS delivery_country;
