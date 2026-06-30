-- Add COD gateway and PENDING_COD status.
-- PostgreSQL CHECK constraints cannot be altered in-place; we drop and recreate them.

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_status_check;
ALTER TABLE payments ADD CONSTRAINT payments_status_check
    CHECK (status IN (
        'PENDING',
        'PENDING_COD',
        'AUTHORISED',
        'CAPTURED',
        'FAILED',
        'REFUNDED',
        'PARTIALLY_REFUNDED'
    ));

-- No constraint on gateway column — it's a free-text label ('fake','stripe','razorpay','cod').

