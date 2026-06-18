DELETE FROM users WHERE email = COALESCE(current_setting('app.admin_email', true), 'admin@zapmarket.local') AND role = 'admin';
