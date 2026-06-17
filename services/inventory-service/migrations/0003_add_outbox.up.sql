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
