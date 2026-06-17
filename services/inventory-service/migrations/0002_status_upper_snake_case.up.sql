-- Migrate reservation status values and constraint to UPPER_SNAKE_CASE.
ALTER TABLE inventory_reservations
    DROP CONSTRAINT inventory_reservations_status_check;

UPDATE inventory_reservations SET status = 'RESERVED'  WHERE status = 'reserved';
UPDATE inventory_reservations SET status = 'CONFIRMED' WHERE status = 'confirmed';
UPDATE inventory_reservations SET status = 'RELEASED'  WHERE status = 'released';

ALTER TABLE inventory_reservations
    ALTER COLUMN status SET DEFAULT 'RESERVED',
    ADD CONSTRAINT inventory_reservations_status_check
        CHECK (status IN ('RESERVED', 'CONFIRMED', 'RELEASED'));

-- Update partial index to match new casing.
DROP INDEX IF EXISTS idx_reservations_expires;
CREATE INDEX idx_reservations_expires ON inventory_reservations (expires_at)
    WHERE status = 'RESERVED';
