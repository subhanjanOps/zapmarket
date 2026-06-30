DROP TABLE IF EXISTS seller_bank_accounts;

ALTER TABLE seller_ledger
    DROP COLUMN IF EXISTS tds_paise,
    DROP COLUMN IF EXISTS gst_on_commission_paise;
