ALTER TABLE payments DROP CONSTRAINT payments_status_check;

UPDATE payments SET status = 'pending'            WHERE status = 'PENDING';
UPDATE payments SET status = 'authorised'         WHERE status = 'AUTHORISED';
UPDATE payments SET status = 'captured'           WHERE status = 'CAPTURED';
UPDATE payments SET status = 'failed'             WHERE status = 'FAILED';
UPDATE payments SET status = 'refunded'           WHERE status = 'REFUNDED';
UPDATE payments SET status = 'partially_refunded' WHERE status = 'PARTIALLY_REFUNDED';

ALTER TABLE payments
    ALTER COLUMN status SET DEFAULT 'pending',
    ADD CONSTRAINT payments_status_check
        CHECK (status IN ('pending', 'authorised', 'captured', 'failed', 'refunded', 'partially_refunded'));

ALTER TABLE refunds DROP CONSTRAINT refunds_status_check;

UPDATE refunds SET status = 'pending'   WHERE status = 'PENDING';
UPDATE refunds SET status = 'processed' WHERE status = 'PROCESSED';
UPDATE refunds SET status = 'failed'    WHERE status = 'FAILED';

ALTER TABLE refunds
    ALTER COLUMN status SET DEFAULT 'pending',
    ADD CONSTRAINT refunds_status_check
        CHECK (status IN ('pending', 'processed', 'failed'));
