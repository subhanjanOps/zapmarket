-- BUG-007: /api/v1/skus was missing from gateway_routes; seller-ui SKU
-- operations returned 404 through the gateway.
-- Also adds /v1/auth/me and image endpoints that were discovered missing.
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix) VALUES
    ('/api/v1/skus',          'product-catalog-service',  'method_split', false),
    ('/api/v1/images',        'product-catalog-service',  'required',     false),
    ('/v1/auth/me',           'auth-service',             'required',     false),
    ('/v1/auth/logout',       'auth-service',             'required',     false),
    ('/v1/admin/orders',      'order-management-service', 'required',     false),
    ('/v1/admin',             'auth-service',             'required',     false)
ON CONFLICT (path_prefix) DO NOTHING;
