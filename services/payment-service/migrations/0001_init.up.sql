CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- One payment attempt per order. An order can have multiple rows here
-- (retries, failed attempts).
CREATE TABLE IF NOT EXISTS payments (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         UUID         NOT NULL,
    user_id          UUID         NOT NULL,
    idempotency_key  UUID         NOT NULL UNIQUE,
    status           VARCHAR(50)  NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN (
                         'PENDING',
                         'AUTHORISED',
                         'CAPTURED',
                         'FAILED',
                         'REFUNDED',
                         'PARTIALLY_REFUNDED'
                     )),
    amount           BIGINT       NOT NULL, -- smallest currency unit (paise for INR)
    currency         CHAR(3)      NOT NULL DEFAULT 'INR',
    gateway          VARCHAR(50)  NOT NULL, -- 'fake', 'razorpay', 'stripe'
    gateway_txn_id   VARCHAR(255) UNIQUE,
    gateway_response JSONB,
    failure_reason   TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payments_order  ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status) WHERE deleted_at IS NULL;

-- GST breakdown per payment. Not populated by this stage's ChargeCard flow
-- (no tax calculation logic exists yet) but the schema is authoritative
-- per db-design.md and other services may write to it later.
CREATE TABLE IF NOT EXISTS payment_taxes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id  UUID        NOT NULL REFERENCES payments (id),
    tax_type    VARCHAR(50) NOT NULL CHECK (tax_type IN ('CGST', 'SGST', 'IGST', 'CESS')),
    rate_bps    INT         NOT NULL,
    base_amount BIGINT      NOT NULL,
    tax_amount  BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Double-entry ledger — every money movement has two rows (debit + credit).
-- Immutable — no updated_at / deleted_at.
CREATE TABLE IF NOT EXISTS ledger_entries (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id  UUID         NOT NULL REFERENCES payments (id),
    entry_type  VARCHAR(50)  NOT NULL CHECK (entry_type IN ('debit', 'credit')),
    account     VARCHAR(100) NOT NULL, -- 'revenue', 'accounts_receivable', 'refund', ...
    amount      BIGINT       NOT NULL CHECK (amount > 0),
    currency    CHAR(3)      NOT NULL DEFAULT 'INR',
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_payment ON ledger_entries (payment_id);
CREATE INDEX IF NOT EXISTS idx_ledger_account ON ledger_entries (account);

-- Refunds linked to a captured payment.
CREATE TABLE IF NOT EXISTS refunds (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id        UUID        NOT NULL REFERENCES payments (id),
    order_id          UUID        NOT NULL,
    amount            BIGINT      NOT NULL,
    currency          CHAR(3)     NOT NULL DEFAULT 'INR',
    reason            VARCHAR(255),
    status            VARCHAR(50) NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED')),
    gateway_refund_id VARCHAR(255),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refunds_payment ON refunds (payment_id);
CREATE INDEX IF NOT EXISTS idx_refunds_order   ON refunds (order_id);

-- Outbox for payment events (same Debezium CDC pattern as order service).
-- Not drained by anything yet — Kafka/Debezium activation is Stage 7. Rows
-- just accumulate with published_at = NULL until then.
CREATE TABLE IF NOT EXISTS outbox (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id   UUID         NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type     VARCHAR(100) NOT NULL,
    payload        JSONB        NOT NULL,
    published_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
