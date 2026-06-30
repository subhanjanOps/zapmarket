# Plan 12 — Razorpay Gateway

**Effort:** M | **Impact:** H | **Depends on:** Plan 05 (COD done first to avoid schema conflicts)

## Context
`gateway` column already allows `'razorpay'` as a value. No Go implementation exists. Razorpay is the primary India payment gateway — higher acceptance rates than Stripe for Indian cards, UPI, netbanking, and wallets. Stripe processes India payments with ~15% higher failure rates due to 2FA/AFA compliance differences.

## Scope
- Implement `RazorpayPaymentGateway` satisfying the existing `PaymentGateway` interface
- Wire via `PAYMENT_GATEWAY` env var (same pattern as carrier in Plan 06)
- Support: card charge, UPI collect, webhook verification

## Tasks
- [ ] Add `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`, `RAZORPAY_WEBHOOK_SECRET` to payment-service `.env.example`
- [ ] Create `internal/gateway/razorpay/client.go`
- [ ] Implement `Charge(ctx, req)`: create Razorpay Order (`POST /v1/orders`), return `order_id` as charge reference; actual capture happens via frontend checkout JS + webhook
- [ ] Implement `Capture(ctx, paymentID)`: call `POST /v1/payments/{id}/capture`
- [ ] Implement `Refund(ctx, paymentID, amountPaise)`: call `POST /v1/payments/{id}/refund`
- [ ] Add `POST /webhooks/razorpay`: verify HMAC-SHA256 signature (`X-Razorpay-Signature` header), handle `payment.captured` and `payment.failed` events
- [ ] Add `PAYMENT_GATEWAY` env var to payment-service config (`stripe` | `razorpay`); factory returns correct impl
- [ ] Handle UPI-specific flow: Razorpay orders for UPI require polling or webhook; ensure idempotency key passed as `receipt` field

## Done criteria
- `PAYMENT_GATEWAY=razorpay` processes a test payment end-to-end
- Webhook signature verification rejects tampered payloads
- Refund works via Razorpay API
- `PAYMENT_GATEWAY=stripe` still works (no regression)
- Idempotency preserved: duplicate webhook fires are no-ops
