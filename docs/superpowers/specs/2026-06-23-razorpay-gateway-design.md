# Razorpay Payment Gateway Integration

**Date:** 2026-06-23
**Status:** Approved
**Service:** `services/payment-service`

---

## Overview

Replace `FakePaymentGateway` with a Razorpay implementation of the existing `PaymentGateway` interface. The interface contract and webhook handler are already correct — Razorpay uses HMAC-SHA256-over-body signatures, which is exactly what `webhook_handler.go` already verifies. No webhook changes are needed.

When `RAZORPAY_KEY_ID` is set in the environment the service uses `RazorpayGateway`; when absent it falls back to `FakePaymentGateway` so local development requires no Razorpay credentials.

---

## Interface (unchanged)

```go
// Already defined in internal/domain/contracts/repositories.go
type ChargeResult struct {
    GatewayTxnID string
}

type PaymentGateway interface {
    Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID) (*ChargeResult, error)
    Refund(ctx context.Context, gatewayTxnID string, amount int64, currency string) (refundRef string, err error)
}
```

---

## Razorpay API mapping

### Charge → Razorpay Orders + Payments flow

Razorpay's payment model is two-step:
1. Create a Razorpay **Order** (`POST /v1/orders`) — returns `razorpay_order_id`
2. The frontend collects card details and calls Razorpay's JS SDK to **capture** the payment — Razorpay sends a webhook on success/failure

For server-to-server (no browser), use **Razorpay's direct charge via test mode**:
- `POST /v1/payments/create/json` with card details (test only) — not for production
- In production: `Charge` creates the Razorpay Order and returns the `order_id` as `GatewayTxnID`; the actual capture happens asynchronously via webhook

**Implementation decision:** `Charge` creates a Razorpay Order and immediately attempts auto-capture via `POST /v1/payments/create/json` in test mode. In production mode (`APP_ENV=production`) it creates the order only and returns; the payment is captured by the buyer's browser + webhook. The `GatewayTxnID` returned is the Razorpay `payment_id` (from auto-capture in test) or `order_id` (in production, pending webhook).

This means `HandleCaptureWebhook` in `payment_service.go` is the production path for marking a payment captured — it is already implemented.

### Refund → `POST /v1/payments/:id/refund`

Razorpay refund API:
```
POST https://api.razorpay.com/v1/payments/{payment_id}/refund
Body: { "amount": <amount_in_paise> }
```
Returns `{ "id": "rfnd_..." }`. The refund reference (`rfnd_...`) is stored as the refund's `gateway_refund_id`.

---

## New file: `internal/gateway/razorpay_gateway.go`

```go
type RazorpayGateway struct {
    keyID     string
    keySecret string
    baseURL   string      // https://api.razorpay.com/v1 (overridable for tests)
    client    *http.Client
    logger    *slog.Logger
}
```

### `Charge` implementation
1. `POST /v1/orders` with `{ amount, currency, receipt: idempotencyKey.String() }`
2. In test mode (`RAZORPAY_TEST_MODE=true`): immediately call `POST /v1/payments/create/json` with test card `4111111111111111`; return `payment_id` as `GatewayTxnID`
3. In production mode: return the `order_id` as `GatewayTxnID`; payment captured asynchronously by webhook
4. Auth: HTTP Basic with `keyID:keySecret`

### `Refund` implementation
1. `POST /v1/payments/{gatewayTxnID}/refund` with `{ amount }`
2. Return `refund.id` as `refundRef`

### Error mapping
| Razorpay error code | Mapped to |
|---|---|
| `BAD_REQUEST_ERROR` / `payment_failed` | `pkgerrors.NewConflict("PAYMENT_FAILED", msg)` |
| Network timeout | `pkgerrors.NewInternal("GATEWAY_TIMEOUT", msg)` |
| `GATEWAY_ERROR` | `pkgerrors.NewInternal("GATEWAY_ERROR", msg)` |
| 4xx other | `pkgerrors.NewValidation("GATEWAY_VALIDATION", msg)` |

---

## Config (env vars)

| Var | Required | Default | Description |
|---|---|---|---|
| `RAZORPAY_KEY_ID` | Yes (to enable) | — | Razorpay key ID |
| `RAZORPAY_KEY_SECRET` | Yes | — | Razorpay key secret |
| `RAZORPAY_TEST_MODE` | No | `false` | Auto-capture in test mode |
| `PAYMENT_WEBHOOK_SECRET` | Yes | — | Already exists; Razorpay uses the same HMAC-SHA256 scheme |

---

## Wire-up in `main.go`

```go
var gw contracts.PaymentGateway
if cfg.RazorpayKeyID != "" {
    gw = gateway.NewRazorpayGateway(cfg.RazorpayKeyID, cfg.RazorpayKeySecret, cfg.RazorpayTestMode, log)
    log.Info("using Razorpay payment gateway", "test_mode", cfg.RazorpayTestMode)
} else {
    gw = gateway.NewFakePaymentGateway(log)
    log.Info("RAZORPAY_KEY_ID not set, using fake payment gateway")
}
```

---

## Webhook: no changes needed

`webhook_handler.go` already:
- Reads `X-Webhook-Signature` and verifies HMAC-SHA256 — matches Razorpay's `X-Razorpay-Signature`
- Reads `X-Webhook-Timestamp` (added in the last session) — matches Razorpay's timestamp header

The only wiring needed: in production, configure the Razorpay dashboard webhook URL to point to `https://<domain>/v1/payment/webhook` and set `PAYMENT_WEBHOOK_SECRET` to the Razorpay webhook secret.

Note: Razorpay sends the signature in `X-Razorpay-Signature`, not `X-Webhook-Signature`. Add a header alias check in `validSignature`: accept either header name.

---

## docker-compose env vars (payment-service)

```yaml
- RAZORPAY_KEY_ID=${RAZORPAY_KEY_ID:-}
- RAZORPAY_KEY_SECRET=${RAZORPAY_KEY_SECRET:-}
- RAZORPAY_TEST_MODE=${RAZORPAY_TEST_MODE:-false}
```

Local development: leave empty → FakePaymentGateway. CI/staging: set test credentials + `RAZORPAY_TEST_MODE=true`.

---

## pkg/config additions

Add to `payment-service`'s `pkg/config/config.go`:
```go
RazorpayKeyID     string
RazorpayKeySecret string
RazorpayTestMode  bool
```

---

## Testing

- Unit test `RazorpayGateway.Charge` and `Refund` with an `httptest.Server` that mocks the Razorpay API
- Test: successful charge in test mode → `GatewayTxnID` is a payment ID
- Test: failed charge → typed conflict error returned
- Test: refund → refund reference returned
- Test: network timeout → typed internal error returned
- Existing `FakePaymentGateway` tests are unchanged

---

## No schema changes

No new database tables or migrations. No new Kafka topics. No changes to the payment repository or service layer.
