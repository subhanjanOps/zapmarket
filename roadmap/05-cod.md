# Plan 05 — Cash on Delivery (COD)

**Effort:** M | **Impact:** H | **Depends on:** Plans 01, 04

## Context
COD is absent from every layer: not a valid `gateway` value in payment schema, not in order domain, not in UI. COD accounts for ~60% of Indian e-commerce volume. Without it, ZapMarket cannot serve tier-2/3 cities or first-time buyers.

COD flow differs from prepaid: payment is collected by the delivery agent at doorstep, then reconciled centrally. Payment-service must be notified by logistics-service (via `shipment.delivered` event) rather than by a payment gateway webhook.

## Scope
- Payment schema: add `COD` gateway value
- Order creation: skip Stripe/Razorpay call for COD; mark payment `PENDING_COD`
- Logistics: `shipment.delivered` event triggers payment status update to `CAPTURED` for COD orders
- Settlement: COD amounts go into seller ledger same as prepaid
- COD reconciliation table for agent cash handover tracking

## Out of scope
- COD remittance to platform (cash-in-transit logistics — physical process, not digital)
- COD availability by pincode (can be added as a config later)

## Tasks

### payment-service
- [ ] Migration: add `'cod'` to the `gateway` CHECK constraint on `payments`
- [ ] Add `PENDING_COD` to payment status enum (payment created but not yet collected)
- [ ] Add Kafka consumer for `shipment.delivered`: if order payment has `gateway = 'cod'` and status `PENDING_COD`, update to `CAPTURED`
- [ ] Skip gateway charge call in `ChargeCard` when `gateway = 'cod'`; write ledger entry immediately as `DEBIT_SALE` on delivery confirmation instead

### order-management-service
- [ ] Accept `payment_method: "cod"` in order creation request body
- [ ] If COD: create payment row with `gateway = 'cod'`, `status = 'PENDING_COD'`, skip `checkout.requested` Kafka event (no Stripe charge needed)
- [ ] Publish `order.confirmed` immediately for COD (buyer needs confirmation; payment happens later)

### logistics-service
- [ ] Add `cod_reconciliations` table: (`id UUID PK`, `shipment_id UUID FK`, `agent_id UUID FK delivery_agents`, `amount_paise BIGINT`, `collected_at TIMESTAMPTZ`, `remitted_at TIMESTAMPTZ`, `status TEXT DEFAULT 'COLLECTED'`)
- [ ] `POST /v1/shipments/{id}/deliver` (from Plan 04): if COD order, write `cod_reconciliations` row on delivery confirmation

### notification-service
- [ ] Add consumer for `order.confirmed` on COD orders to send "Your COD order is confirmed" notification

## Done criteria
- COD order creation succeeds without calling Stripe
- Payment status is `PENDING_COD` until delivery
- On `shipment.delivered`, payment transitions to `CAPTURED`
- `cod_reconciliations` row created on delivery
- Seller ledger credited for COD the same as prepaid
