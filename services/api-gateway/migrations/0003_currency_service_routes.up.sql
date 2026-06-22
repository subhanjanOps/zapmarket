-- Register currency-service routes. Public GET endpoints use auth_mode=none;
-- the admin toggle endpoint uses auth_mode=required so the gateway validates
-- the JWT and forwards X-User-Role; the currency-service handler enforces admin.
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix) VALUES
    ('/api/v1/currencies',        'currency-service', 'none',     false),
    ('/api/v1/admin/currencies',  'currency-service', 'required', false)
ON CONFLICT (path_prefix) DO UPDATE
  SET upstream  = EXCLUDED.upstream,
      auth_mode = EXCLUDED.auth_mode;
