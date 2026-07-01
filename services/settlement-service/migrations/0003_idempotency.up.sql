-- Prevent Kafka at-least-once redelivery from double-crediting/debiting the
-- seller ledger for the same payment event.
CREATE UNIQUE INDEX IF NOT EXISTS idx_seller_ledger_payment_entry
    ON seller_ledger (payment_id, entry_type)
    WHERE payment_id IS NOT NULL;

-- Prevent more than one payout from being in flight for a seller at once,
-- across concurrent scheduler replicas.
CREATE UNIQUE INDEX IF NOT EXISTS idx_seller_payouts_seller_pending
    ON seller_payouts (seller_id)
    WHERE status = 'PENDING';
