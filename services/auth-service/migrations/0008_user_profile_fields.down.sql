ALTER TABLE users
    DROP COLUMN IF EXISTS dob,
    DROP COLUMN IF EXISTS gender,
    DROP COLUMN IF EXISTS pfp_url,
    DROP COLUMN IF EXISTS phone_verified,
    DROP COLUMN IF EXISTS terms_accepted_at,
    DROP COLUMN IF EXISTS registration_step;

ALTER TABLE seller_profiles
    DROP COLUMN IF EXISTS business_type,
    DROP COLUMN IF EXISTS tax_id,
    DROP COLUMN IF EXISTS biz_line1,
    DROP COLUMN IF EXISTS biz_city,
    DROP COLUMN IF EXISTS biz_state,
    DROP COLUMN IF EXISTS biz_country,
    DROP COLUMN IF EXISTS biz_pincode;
