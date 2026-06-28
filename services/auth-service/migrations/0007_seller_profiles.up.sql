CREATE TABLE IF NOT EXISTS seller_profiles (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    store_name     TEXT        NOT NULL,
    tagline        TEXT        NOT NULL DEFAULT '',
    category       TEXT        NOT NULL,
    gstin          TEXT,
    pan            TEXT,
    business_phone TEXT        NOT NULL,
    city           TEXT        NOT NULL,
    pincode        TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT seller_profiles_user_id_key UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_seller_profiles_user_id ON seller_profiles(user_id);
