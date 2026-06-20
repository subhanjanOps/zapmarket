-- BUG-001/008: Admin users cannot be registered via the public API (by design —
-- this prevents privilege escalation).
--
-- This migration seeds the initial admin using pgcrypto bcrypt.
-- Default credentials: admin@zapmarket.local / Admin@zapmarket1!
-- CHANGE THESE IN PRODUCTION via the admin panel:
--   PUT /v1/admin/users/{id}/role (requires existing admin token)
--
-- To bootstrap with custom credentials, set these Postgres session variables
-- before running migrations:
--   SET app.admin_email = 'your@email.com';
--   SET app.admin_password = 'YourP@ssword1';
INSERT INTO users (
    id,
    email,
    password_hash,
    full_name,
    role,
    is_verified,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    COALESCE(current_setting('app.admin_email', true), 'admin@zapmarket.com'),
    crypt(
        COALESCE(current_setting('app.admin_password', true), 'Admin@zapmarket1!'),
        gen_salt('bf', 10)
    ),
    'System Admin',
    'admin',
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE role = 'admin'
);
