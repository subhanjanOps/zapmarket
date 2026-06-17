CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS gateway_routes (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    path_prefix  TEXT        NOT NULL UNIQUE,
    upstream     TEXT        NOT NULL,
    auth_mode    TEXT        NOT NULL DEFAULT 'required'
                             CHECK (auth_mode IN ('none', 'required', 'method_split')),
    strip_prefix BOOLEAN     NOT NULL DEFAULT false,
    enabled      BOOLEAN     NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_gateway_routes_enabled ON gateway_routes (enabled) WHERE enabled = true;

-- Trigger to update updated_at on any row change.
CREATE OR REPLACE FUNCTION gateway_routes_set_updated_at()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER gateway_routes_updated_at
    BEFORE UPDATE ON gateway_routes
    FOR EACH ROW EXECUTE FUNCTION gateway_routes_set_updated_at();

-- NOTIFY trigger so the gateway watcher rebuilds the router instantly.
CREATE OR REPLACE FUNCTION gateway_routes_notify()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_notify('gateway_route_changed', TG_OP);
    RETURN NULL;
END;
$$;

CREATE TRIGGER gateway_routes_changed
    AFTER INSERT OR UPDATE OR DELETE ON gateway_routes
    FOR EACH STATEMENT EXECUTE FUNCTION gateway_routes_notify();

CREATE TABLE IF NOT EXISTS gateway_audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    ts          TIMESTAMPTZ NOT NULL DEFAULT now(),
    request_id  TEXT,
    user_id     TEXT,
    ip          TEXT,
    method      TEXT,
    path        TEXT,
    upstream    TEXT,
    status_code INT,
    event       TEXT        NOT NULL,
    detail      TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_ts     ON gateway_audit_log (ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_userid ON gateway_audit_log (user_id, ts DESC) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_event  ON gateway_audit_log (event, ts DESC);

-- Seed the initial routes so the gateway works out-of-the-box.
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix) VALUES
    ('/v1/auth/register',  'auth-service',                'none',         false),
    ('/v1/auth/login',     'auth-service',                'none',         false),
    ('/v1/auth/refresh',   'auth-service',                'none',         false),
    ('/v1/auth/oauth',     'auth-service',                'none',         false),
    ('/v1/auth',           'auth-service',                'required',     false),
    ('/api/v1/categories', 'product-catalog-service',     'none',         false),
    ('/api/v1/products',   'product-catalog-service',     'method_split', false),
    ('/v1/orders',         'order-management-service',    'required',     false)
ON CONFLICT (path_prefix) DO NOTHING;
