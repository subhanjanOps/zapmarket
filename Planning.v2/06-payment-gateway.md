# Phase 6 — Payment Gateway Integration

**Goal:** Replace `FakePaymentGateway` with a real payment processor (Razorpay recommended for INR; Stripe as alternative for multi-currency). Wire the full async webhook lifecycle.

**Current state:** `FakePaymentGateway` resolves every charge synchronously and always succeeds. `ChargeCard` in `payment-service` assumes synchronous capture. Webhook handlers (`HandleCaptureWebhook`, `HandleFailureWebhook`) exist in the service layer but are never called.

---

## 6.1 Gateway Selection

### Razorpay (recommended)
- Native INR support, UPI, netbanking, cards
- Webhooks for `payment.captured`, `payment.failed`, `refund.processed`
- Go SDK: `github.com/razorpay/razorpay-go`

### Stripe
- Multi-currency, global cards
- Webhooks for `payment_intent.succeeded`, `payment_intent.payment_failed`
- Go SDK: `github.com/stripe/stripe-go/v76`

**Recommendation:** Razorpay for primary market (India/INR), with the `contracts.PaymentGateway` interface allowing Stripe to be added later.

---

## 6.2 Implementation Plan

### 6.2.1 Razorpay Gateway Implementation

```go
// internal/gateway/razorpay.go
type RazorpayGateway struct {
    client *razorpay.Client
    logger *slog.Logger
}

func (g *RazorpayGateway) Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID) (*contracts.ChargeResult, error) {
    // Create a Razorpay Order (not a charge — Razorpay is two-step: order → payment)
    // Returns order_id which the frontend uses to open the Razorpay checkout widget
    // Actual capture happens via webhook after customer completes payment
    order, err := g.client.Order.Create(map[string]interface{}{
        "amount":   amount,
        "currency": currency,
        "receipt":  idempotencyKey.String(),
    }, nil)
    // Return order_id as GatewayTxnID; payment.status stays PENDING
    // until webhook confirms capture
}

func (g *RazorpayGateway) Refund(ctx context.Context, gatewayTxnID string, amount int64, currency string) (string, error) {
    refund, err := g.client.Payment.Refund(gatewayTxnID, map[string]interface{}{"amount": amount}, nil)
    return refund["id"].(string), err
}
```

### 6.2.2 Checkout Flow Change

With a real gateway, the checkout becomes asynchronous:

**Old flow (FakeGateway):**
```
POST /v1/orders → saga → Charge → PaymentCaptured → order CONFIRMED (synchronous)
```

**New flow (Razorpay):**
```
POST /v1/orders → saga → Create Razorpay Order → order PENDING_PAYMENT
Frontend opens Razorpay widget
Customer completes payment
Razorpay sends webhook → POST /v1/webhooks/razorpay
Webhook handler → HandleCaptureWebhook → PaymentCaptured → order CONFIRMED
```

This requires:
1. `order-management-service` saga: after inventory reserve, create Razorpay order and return `gateway_order_id` + `gateway_key` to the client without completing the saga.
2. Saga completion is triggered by the webhook, not inline.
3. Add `PENDING_PAYMENT` order status between `CREATED` and `CONFIRMED`.

### 6.2.3 Webhook Endpoint

```go
// payment-service: POST /v1/webhooks/razorpay
func (h *WebhookHandler) HandleRazorpay(w http.ResponseWriter, r *http.Request) {
    // 1. Verify HMAC-SHA256 signature using PAYMENT_WEBHOOK_SECRET
    body, _ := io.ReadAll(r.Body)
    sig := r.Header.Get("X-Razorpay-Signature")
    if !verifyRazorpaySignature(body, sig, webhookSecret) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    var event razorpayEvent
    json.Unmarshal(body, &event)

    switch event.Event {
    case "payment.captured":
        svc.HandleCaptureWebhook(r.Context(), paymentID, event.Payload.Payment.Entity.ID)
    case "payment.failed":
        svc.HandleFailureWebhook(r.Context(), paymentID, event.Payload.Payment.Entity.ErrorDescription)
    case "refund.processed":
        // update refund status
    }
    w.WriteHeader(http.StatusOK)
}
```

The webhook endpoint is **unauthenticated** (no JWT) but signature-verified. Register it in api-gateway with `auth_mode = "none"`.

### 6.2.4 Webhook Idempotency

Razorpay retries webhooks on non-200 responses. Use Redis `SET NX` with the Razorpay event ID (included in headers) to deduplicate:

```go
key := fmt.Sprintf("webhook:razorpay:%s", eventID)
set, _ := rdb.SetNX(ctx, key, 1, 24*time.Hour).Result()
if !set { w.WriteHeader(http.StatusOK); return } // already processed
```

---

## 6.3 Frontend Integration

### seller-ui checkout (if sellers do B2B payments)

Not in current scope — sellers receive payments from buyers, they don't pay themselves.

### buyer-ui checkout (future `buyer-ui` service)

The checkout page needs to:
1. Call `POST /v1/orders` → receive `{order_id, gateway_order_id, gateway_key, amount, currency}`.
2. Load Razorpay checkout JS SDK.
3. Open the Razorpay modal with the returned `gateway_order_id`.
4. On modal success callback: poll `GET /v1/orders/:id` until status = `CONFIRMED` (or show "payment processing").
5. On modal failure: show error, offer retry.

---

## 6.4 Environment Variables

Add to `payment-service` in `docker-compose.yml` and `.env.example`:

```env
RAZORPAY_KEY_ID=rzp_test_...
RAZORPAY_KEY_SECRET=...
PAYMENT_WEBHOOK_SECRET=...   # already exists
PAYMENT_GATEWAY=razorpay     # or "fake" for local dev
```

Keep `FakePaymentGateway` active when `PAYMENT_GATEWAY=fake` so local development doesn't need Razorpay credentials.

---

## 6.5 Refund Flow Update

With Razorpay, refunds are initiated against the Razorpay payment ID (not the order ID). The `payment-service` already stores `gateway_txn_id` (the Razorpay payment ID after capture). The existing `RefundPayment` service method maps cleanly:

```go
gatewayRefundID, err := s.gateway.Refund(ctx, *payment.GatewayTxnID, amount, payment.Currency)
// → calls razorpay.Payment.Refund(payment_id, amount)
```

No changes needed to the service layer — only the gateway implementation.

---

## Acceptance criteria

- Checkout creates a Razorpay order and returns `gateway_order_id` to the client
- Webhook signature verification rejects tampered payloads (400)
- Duplicate webhook delivery is a no-op (idempotent)
- Successful payment webhook transitions order to `CONFIRMED` and publishes `order.confirmed` event
- Failed payment webhook transitions order to `PAYMENT_FAILED` and releases inventory reservation
- `FakePaymentGateway` still works when `PAYMENT_GATEWAY=fake`
- Refund can be initiated from backoffice-ui admin panel

---

## Estimated effort

3 weeks (Razorpay integration: 1 week; async checkout flow refactor: 1 week; webhook handling + testing: 1 week).
