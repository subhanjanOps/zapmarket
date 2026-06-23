-- Normalise all gateway routes to use /v1/ prefix (drop the /api prefix).
-- Routes for product-catalog, images, and currency were inconsistently seeded
-- with /api/v1/ while auth/orders used /v1/.
UPDATE gateway_routes SET path_prefix = '/v1/categories'       WHERE path_prefix = '/api/v1/categories';
UPDATE gateway_routes SET path_prefix = '/v1/products'         WHERE path_prefix = '/api/v1/products';
UPDATE gateway_routes SET path_prefix = '/v1/skus'             WHERE path_prefix = '/api/v1/skus';
UPDATE gateway_routes SET path_prefix = '/v1/images'           WHERE path_prefix = '/api/v1/images';
UPDATE gateway_routes SET path_prefix = '/v1/currencies'       WHERE path_prefix = '/api/v1/currencies';
UPDATE gateway_routes SET path_prefix = '/v1/admin/currencies' WHERE path_prefix = '/api/v1/admin/currencies';
UPDATE gateway_routes SET path_prefix = '/v1'                  WHERE path_prefix = '/api/v1';
