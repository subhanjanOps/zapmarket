CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE seller_ledger (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id        UUID NOT NULL,
    order_id         UUID,
    payment_id       UUID,
    entry_type       VARCHAR(32) NOT NULL,
    amount_paise     BIGINT NOT NULL,
    commission_paise BIGINT NOT NULL DEFAULT 0,
    net_paise        BIGINT NOT NULL,
    currency         VARCHAR(3) NOT NULL DEFAULT 'INR',
    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seller_balances (
    seller_id       UUID PRIMARY KEY,
    pending_paise   BIGINT NOT NULL DEFAULT 0,
    paid_out_paise  BIGINT NOT NULL DEFAULT 0,
    currency        VARCHAR(3) NOT NULL DEFAULT 'INR',
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seller_payouts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id           UUID NOT NULL,
    amount_paise        BIGINT NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    razorpay_payout_id  TEXT,
    status              VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    initiated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ
);

CREATE INDEX idx_seller_ledger_seller ON seller_ledger(seller_id);
CREATE INDEX idx_seller_payouts_seller ON seller_payouts(seller_id, status);
