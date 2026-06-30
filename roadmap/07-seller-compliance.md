# Plan 07 — Seller Compliance (TDS, GST, Payouts)

**Effort:** M | **Impact:** H | **Depends on:** Plan 01

## Context
settlement-service has commission deduction (BPS-based ledger entry) but is missing legal requirements:
- **TDS (1%)** under Section 194-O of the Income Tax Act — mandatory for all marketplace payments
- **GST on platform commission (18%)** — mandatory under CGST Act
- **Payout bank/UPI details** — no storage for seller bank accounts; payouts cannot be made
- **Payout scheduler** — referenced in a code comment in `main.go:74` but never wired

None of these are optional for an Indian marketplace operating legally.

## Scope
- Add TDS and GST deduction to settlement ledger
- Add `seller_bank_accounts` table
- Wire the weekly payout scheduler
- Add Razorpay Payout API integration for NEFT/UPI disbursement

## Out of scope
- TDS certificate generation (Form 16A) — future
- GST invoice generation — future
- Multi-currency payouts

## Tasks

### Schema
- [ ] Migration: add `tds_paise BIGINT NOT NULL DEFAULT 0`, `gst_on_commission_paise BIGINT NOT NULL DEFAULT 0` columns to settlement `ledger_entries` table
- [ ] Migration: create `seller_bank_accounts` table (`id UUID PK`, `seller_id UUID`, `account_holder_name TEXT`, `account_number TEXT`, `ifsc_code TEXT`, `bank_name TEXT`, `upi_id TEXT`, `is_verified BOOL DEFAULT FALSE`, `is_primary BOOL DEFAULT FALSE`, `created_at TIMESTAMPTZ`)
- [ ] Migration: add `'DEBIT_TDS'`, `'DEBIT_GST_COMMISSION'` to ledger `entry_type` CHECK constraint

### Business logic (settlement-service)
- [ ] In `CreditSaleUseCase`: after computing `commission_paise`, compute `tds_paise = round(gross_sale * 0.01)` and `gst_on_commission_paise = round(commission_paise * 0.18)`
- [ ] Write separate ledger entries for each deduction (audit trail per deduction type)
- [ ] Net seller credit = `sale_amount - commission - tds - gst_on_commission`

### Payout flow
- [ ] Add `POST /v1/sellers/{id}/bank-accounts` and `GET` endpoints
- [ ] Add `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET` to settlement-service `.env.example`
- [ ] Implement `RazorpayPayoutGateway`: call Razorpay Fund Account + Payout API
- [ ] Wire weekly payout scheduler in `main.go`: cron `0 10 * * 1` (every Monday 10am) — query sellers with `net_balance_paise > 0`, call payout gateway, write `DEBIT_PAYOUT` ledger entry
- [ ] Add `return_debit_paise` column to `ledger_entries` and `'DEBIT_RETURN'` entry type for use in Plan 11

## Done criteria
- Every sale ledger entry has correct TDS + GST deduction amounts
- Net seller balance reflects all deductions
- Seller can register a bank account via API
- Weekly scheduler runs and creates payout entries (test with mock gateway in dev)
