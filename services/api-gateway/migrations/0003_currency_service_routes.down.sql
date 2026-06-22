DELETE FROM gateway_routes
WHERE path_prefix IN ('/api/v1/currencies', '/v1/admin/currencies');
