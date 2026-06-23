# Email Notifier — SMTP Implementation

**Date:** 2026-06-23
**Status:** Approved
**Service:** `services/notification-service`

---

## Overview

Replace the `LogNotifier` placeholder with a production-ready SMTP email notifier. The `Notifier` interface already exists and is correct — this spec adds one new implementation behind it. When `SMTP_HOST` is set in the environment the service uses the SMTP notifier; when absent it falls back to `LogNotifier` so local development requires no mail server.

---

## Interface (unchanged)

```go
// Already defined in internal/notifier/notifier.go
type Notification struct {
    UserID    string
    EventType string
    Subject   string
    Body      string
}

type Notifier interface {
    Send(ctx context.Context, n Notification) error
}
```

---

## New file: `internal/notifier/smtp_notifier.go`

### Responsibilities
1. Resolve the user's email address from `UserID` via a gRPC call to auth-service (`GetUser` or `ValidateToken` — use whatever exposes email by user ID)
2. Compose a plain-text + HTML email using Go's `html/template`
3. Dial the SMTP server and send via `net/smtp`
4. Return a typed error on failure so the Kafka consumer retries

### Config (env vars)
| Var | Required | Default | Description |
|---|---|---|---|
| `SMTP_HOST` | Yes (to enable) | — | e.g. `smtp.sendgrid.net` |
| `SMTP_PORT` | No | `587` | SMTP port (STARTTLS) |
| `SMTP_USER` | Yes | — | SMTP username |
| `SMTP_PASSWORD` | Yes | — | SMTP password / API key |
| `SMTP_FROM` | No | `noreply@zapmarket.com` | From address |
| `AUTH_SERVICE_ADDR` | Yes | `localhost:50051` | To resolve user email |

### Email lookup
`SMTPNotifier` holds a gRPC `AuthServiceClient`. On `Send`, it calls `GetUserByID(userID)` to retrieve the email address. If the user is not found the notification is dropped (logged as warn, not an error — deleted users should not block the consumer).

### HTML email template
Inline Go `html/template` string — no external files. Template variables: `Subject`, `Body`, `Year`. Output is a simple transactional layout: ZapMarket wordmark header, body text, footer with unsubscribe note.

### Retry behaviour
`Send` returns an error only on SMTP dial/auth failure or a `5xx` response code. The Kafka consumer already retries on error (`return err` in `Handle`). Soft bounces (user not found, template error) are swallowed after logging — they must not cause infinite retry loops.

---

## Wire-up in `main.go`

```go
var n notifier.Notifier
if cfg.SMTPHost != "" {
    authConn, _ := grpc.NewClient(cfg.AuthServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    n = notifier.NewSMTPNotifier(cfg, authConn, log)
    log.Info("using SMTP notifier", "host", cfg.SMTPHost)
} else {
    n = notifier.NewLogNotifier(log)
    log.Info("SMTP_HOST not set, using log notifier")
}
```

---

## Auth service: GetUserByID gRPC RPC

The auth-service proto already exposes `ValidateToken`. Check whether `GetUserByID` exists in the proto; if not, add a new RPC:

```protobuf
rpc GetUserByID(GetUserByIDRequest) returns (UserResponse);
message GetUserByIDRequest { string user_id = 1; }
```

This requires regenerating the proto, copying the generated files to `services/notification-service/proto/authpb/`, and adding the gRPC client to the notification-service.

If adding a new RPC is undesirable, an alternative is a lightweight HTTP call to `GET /v1/admin/users/:id` on the auth-service. The SMTP notifier would hold an `http.Client` and the auth-service's internal HTTP URL. This avoids proto changes at the cost of HTTP overhead.

**Decision: use the HTTP fallback** — no proto changes, simpler to implement, acceptable latency for async notification delivery.

---

## Email templates per event type

| Event | Subject | Body summary |
|---|---|---|
| `order.confirmed` | "Your order has been confirmed" | Order ID, thank-you message |
| `order.cancelled` | "Your order has been cancelled" | Order ID, support CTA |
| `payment.captured` | "Payment successful" | Amount, order ID |
| `payment.failed` | "Payment failed" | Order ID, retry CTA |
| `payment.refunded` | "Refund processed" | Amount, timeline (3-5 days) |
| `inventory.depleted` | "Stock depleted for your product" | SKU ID, restock CTA (sent to seller) |

The `Body` field on `Notification` is already populated by `consumer/handler.go` — the SMTP notifier uses it verbatim as the email body text. The HTML template wraps it in the branded layout.

---

## docker-compose env vars (notification-service)

Add to the existing `notification-service` environment block:

```yaml
- SMTP_HOST=${SMTP_HOST:-}
- SMTP_PORT=${SMTP_PORT:-587}
- SMTP_USER=${SMTP_USER:-}
- SMTP_PASSWORD=${SMTP_PASSWORD:-}
- SMTP_FROM=${SMTP_FROM:-noreply@zapmarket.com}
- AUTH_SERVICE_ADDR=zapmarket-auth-service:50051
- AUTH_SERVICE_HTTP_URL=http://zapmarket-auth-service:8080
```

Leaving `SMTP_HOST` empty means local `docker compose up` uses `LogNotifier` without any configuration.

---

## Testing

- Unit test `SMTPNotifier.Send` with a mock SMTP server (`github.com/emersion/go-smtp` test helper or a simple `net.Listener` mock)
- Test: user not found → notification dropped, no error returned
- Test: SMTP dial failure → error returned (consumer retries)
- Test: successful send → dedup key set in Redis (existing consumer test coverage)

---

## No schema changes

No new database tables or migrations. No new Kafka topics. No changes to the consumer handler.
