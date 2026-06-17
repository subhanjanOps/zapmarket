DROP INDEX IF EXISTS idx_reservations_expires;

ALTER TABLE inventory_reservations
    DROP CONSTRAINT inventory_reservations_status_check;

UPDATE inventory_reservations SET status = 'reserved'  WHERE status = 'RESERVED';
UPDATE inventory_reservations SET status = 'confirmed' WHERE status = 'CONFIRMED';
UPDATE inventory_reservations SET status = 'released'  WHERE status = 'RELEASED';

ALTER TABLE inventory_reservations
    ALTER COLUMN status SET DEFAULT 'reserved',
    ADD CONSTRAINT inventory_reservations_status_check
        CHECK (status IN ('reserved', 'confirmed', 'released'));

CREATE INDEX idx_reservations_expires ON inventory_reservations (expires_at)
    WHERE status = 'reserved';
