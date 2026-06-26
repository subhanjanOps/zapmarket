-- Revert ledger entry_type values back to lowercase.
UPDATE ledger_entries SET entry_type = 'debit'  WHERE entry_type = 'DEBIT';
UPDATE ledger_entries SET entry_type = 'credit' WHERE entry_type = 'CREDIT';
