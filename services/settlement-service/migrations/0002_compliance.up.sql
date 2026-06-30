-- TDS and GST deduction columns on ledger entries
ALTER TABLE seller_ledger
    ADD COLUMN IF NOT EXISTS tds_paise              BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gst_on_commission_paise BIGINT NOT NULL DEFAULT 0;

-- Seller bank accounts for payout
CREATE TABLE seller_bank_accounts (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id            UUID NOT NULL,
    account_holder_name  TEXT NOT NULL,
    account_number       TEXT NOT NULL,
    ifsc_code            TEXT NOT NULL,
    bank_name            TEXT NOT NULL,
    upi_id               TEXT,
    is_verified          BOOL NOT NULL DEFAULT FALSE,
    is_primary           BOOL NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seller_bank_seller ON seller_bank_accounts(seller_id);
