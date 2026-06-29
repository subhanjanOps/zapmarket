CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE coupons (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(64) NOT NULL UNIQUE,
    discount_type   VARCHAR(32) NOT NULL,
    discount_value  BIGINT NOT NULL,
    min_order_paise BIGINT NOT NULL DEFAULT 0,
    max_uses_total  INT NOT NULL DEFAULT 0,
    max_uses_per_user INT NOT NULL DEFAULT 0,
    expires_at      TIMESTAMPTZ,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE coupon_usage (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coupon_id  UUID NOT NULL REFERENCES coupons(id),
    user_id    UUID NOT NULL,
    order_id   UUID NOT NULL,
    used_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_coupon_usage_coupon ON coupon_usage(coupon_id);
CREATE INDEX idx_coupon_usage_user ON coupon_usage(coupon_id, user_id);
