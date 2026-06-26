-- Migrate ledger entry_type values to UPPER_SNAKE_CASE.
UPDATE ledger_entries SET entry_type = 'DEBIT'  WHERE entry_type = 'debit';
UPDATE ledger_entries SET entry_type = 'CREDIT' WHERE entry_type = 'credit';
