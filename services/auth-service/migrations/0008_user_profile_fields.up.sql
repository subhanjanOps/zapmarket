ALTER TABLE users
    ADD COLUMN IF NOT EXISTS dob                DATE,
    ADD COLUMN IF NOT EXISTS gender             VARCHAR(20),
    ADD COLUMN IF NOT EXISTS pfp_url            TEXT,
    ADD COLUMN IF NOT EXISTS phone_verified     BOOLEAN      NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS terms_accepted_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS registration_step  SMALLINT     NOT NULL DEFAULT 4;
-- existing accounts default to step 4 (completed) so they aren't locked out

ALTER TABLE seller_profiles
    ADD COLUMN IF NOT EXISTS business_type  VARCHAR(30) NOT NULL DEFAULT 'individual',
    ADD COLUMN IF NOT EXISTS tax_id         TEXT,
    ADD COLUMN IF NOT EXISTS biz_line1      TEXT,
    ADD COLUMN IF NOT EXISTS biz_city       VARCHAR(100),
    ADD COLUMN IF NOT EXISTS biz_state      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS biz_country    CHAR(2),
    ADD COLUMN IF NOT EXISTS biz_pincode    VARCHAR(20);
