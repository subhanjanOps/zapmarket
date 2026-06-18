DELETE FROM gateway_routes WHERE path_prefix IN (
    '/api/v1/skus', '/api/v1/images', '/v1/auth/me', '/v1/auth/logout', '/v1/admin'
);
