DROP TABLE IF EXISTS cod_reconciliations;

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_status_check;
ALTER TABLE payments ADD CONSTRAINT payments_status_check
    CHECK (status IN (
        'PENDING',
        'AUTHORISED',
        'CAPTURED',
        'FAILED',
        'REFUNDED',
        'PARTIALLY_REFUNDED'
    ));
