-- Route /v1/users/me/* to auth-service for user preference endpoints.
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
VALUES ('/v1/users/me', 'auth-service', 'required', false)
ON CONFLICT (path_prefix) DO NOTHING;
