DROP INDEX IF EXISTS idx_orders_saga_status;
ALTER TABLE orders DROP COLUMN IF EXISTS saga_status;
