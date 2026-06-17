-- Migrate payment and refund status values and constraints to UPPER_SNAKE_CASE.
ALTER TABLE payments DROP CONSTRAINT payments_status_check;

UPDATE payments SET status = 'PENDING'            WHERE status = 'pending';
UPDATE payments SET status = 'AUTHORISED'         WHERE status = 'authorised';
UPDATE payments SET status = 'CAPTURED'           WHERE status = 'captured';
UPDATE payments SET status = 'FAILED'             WHERE status = 'failed';
UPDATE payments SET status = 'REFUNDED'           WHERE status = 'refunded';
UPDATE payments SET status = 'PARTIALLY_REFUNDED' WHERE status = 'partially_refunded';

ALTER TABLE payments
    ALTER COLUMN status SET DEFAULT 'PENDING',
    ADD CONSTRAINT payments_status_check
        CHECK (status IN ('PENDING', 'AUTHORISED', 'CAPTURED', 'FAILED', 'REFUNDED', 'PARTIALLY_REFUNDED'));

ALTER TABLE refunds DROP CONSTRAINT refunds_status_check;

UPDATE refunds SET status = 'PENDING'   WHERE status = 'pending';
UPDATE refunds SET status = 'PROCESSED' WHERE status = 'processed';
UPDATE refunds SET status = 'FAILED'    WHERE status = 'failed';

ALTER TABLE refunds
    ALTER COLUMN status SET DEFAULT 'PENDING',
    ADD CONSTRAINT refunds_status_check
        CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED'));
