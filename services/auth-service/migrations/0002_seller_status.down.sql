DROP INDEX IF EXISTS idx_users_seller_status;
ALTER TABLE users DROP COLUMN IF EXISTS seller_status;
