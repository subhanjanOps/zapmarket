CREATE TABLE IF NOT EXISTS user_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID,
    session_id   TEXT,
    event_type   TEXT NOT NULL CHECK (event_type IN ('view','cart_add','purchase','search')),
    product_id   UUID,
    category_id  UUID,
    metadata     JSONB,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_events_user_cat ON user_events (user_id, category_id, occurred_at DESC)
    WHERE user_id IS NOT NULL AND category_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_events_product ON user_events (product_id, occurred_at DESC)
    WHERE product_id IS NOT NULL;
