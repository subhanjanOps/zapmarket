# Stage 4 — Payment Service

**Status: ✅ Complete (2026-06-16)**

Corresponds to checklist **Phase 10**.

## Goal

Build `payment-service` from scaffold into a working gRPC service that can charge a
card idempotently and record a double-entry ledger — the second dependency Order
Management's saga needs.

## Preconditions
- Stage 1 merged (contracts pattern, migrations).
- Stage 3 not a hard dependency (Payment doesn't call Inventory), but build in parallel
  conceptually so Stage 5 has both ready at once.

## Tasks

### 4.1 Scaffolding
- [x] Same layout convention as Stage 3 (root `main.go`, not `cmd/main.go`):
      `internal/domain`, `internal/domain/contracts`, `internal/repository`,
      `internal/gateway` (concrete `PaymentGateway` implementations), `internal/service`,
      `internal/handler/grpc`, `internal/handler/http`, `migrations/`, `pkg/config` wiring.
- [x] `.env.example` with `DB_NAME=payment`, `HTTP_PORT=8083`, `GRPC_PORT=50054` (next
      free ports after inventory's 8082/50053), and `PAYMENT_WEBHOOK_SECRET` as a
      placeholder. No `PAYMENT_GATEWAY_API_KEY` — see 4.5, no real gateway this stage.

### 4.2 Domain model
- [x] Used the **authoritative** schema from `db-design.md` §5 (same approach as Stage
      3's inventory schema) rather than the simplified `transactions` table this plan
      originally sketched: `payments` (not `transactions` — matches db-design.md's
      naming), `payment_taxes`, `ledger_entries`, `refunds`, `outbox`. Status enum is the
      schema's real one (`pending|authorised|captured|failed|refunded|partially_refunded`),
      not the plan's invented `PENDING`/`SUCCEEDED`/`FAILED`/`REFUNDED`.
- [x] Migration `0001_init.up/down.sql`, and trimmed the now-redundant copy of this
      schema out of `docker-entrypoint-initdb.d/init.sql` (same pattern as Stages 1 & 3).
- [x] `payment_taxes` table exists per the schema but nothing writes to it yet — no tax
      calculation logic exists in this stage. Flagged as a gap, not silently dropped.

### 4.3 Idempotency (without Redis yet)
- [x] `payments.idempotency_key` is `UUID NOT NULL UNIQUE`. `ChargeCard` checks for an
      existing row by key first and returns it unchanged (even if its prior attempt
      failed) rather than re-attempting the gateway call — verified live with two
      identical `ChargeCard` calls returning the same `payment_id` and confirmed only one
      row + one debit/credit ledger pair exists in Postgres.

### 4.4 Proto contract
- [x] `pkg/proto/payment/payment.proto` (flat package, no `v1/` subdirectory — matches
      the existing `auth`/`catalog`/`inventory` packages' actual layout, not the plan's
      `v1/` sketch): `ChargeCard`, `RefundPayment`, `GetTransaction`.
- [x] Generated via `protoc --go_out=. --go-grpc_out=.`, same manual nested-directory
      move as Stage 3's inventory proto required.

### 4.5 Gateway integration
- [x] **Decision (asked, not guessed)**: user chose "Fake only for now" — no real
      Razorpay/Stripe SDK wired this stage; defer until there's an actual gateway account
      with real credentials.
- [x] `contracts.PaymentGateway` interface (`Charge`, `Refund`) in `internal/domain/contracts`.
- [x] `internal/gateway.FakePaymentGateway`: succeeds for any amount except one ending in
      `13` in the smallest currency unit (e.g. paise amount `...13`), which simulates a
      card decline — verified live with amount `5013` → `status: failed`, persisted
      correctly and retrievable via `GetTransaction`.

### 4.6 Webhook handling
- [x] `POST /webhooks/payment` in `internal/handler/http`: verifies `X-Webhook-Signature`
      (hex HMAC-SHA256 over the raw body, the same scheme Razorpay/Stripe both use) before
      trusting anything in the payload, then dispatches to idempotent
      `HandleCaptureWebhook`/`HandleFailureWebhook` service methods. Verified live: valid
      signature → `200` (no-op since FakePaymentGateway already captured synchronously);
      invalid signature → `401`.
  - Note: since FakePaymentGateway resolves synchronously inside `ChargeCard`, nothing in
    this stage's actual flow calls this endpoint — it exists ready for when a real async
    gateway is wired in later, and was verified by hand-computing a valid signature.

## Out of scope
- Kafka publishing of `payment.processed` / `payment.failed` — Stage 7.
- Redis idempotency cache — Stage 9 (Postgres unique constraint covers correctness until then).
- Refund saga / compensation flows beyond a basic `RefundPayment` RPC — full refund
  workflow UI/process is not in the current checklist scope.

## Definition of done
- [x] Service starts a gRPC server on the configured port. Verified via Docker
  (`docker compose up payment-service` → `healthy`, logs show `starting gRPC server
  port=50054`).
- [x] Two identical `ChargeCard` calls with the same `idempotency_key` return the same
  `payment_id` and do not create two ledger entries. Verified live via `grpcurl` +
  direct Postgres query (1 payment row, 2 ledger rows total — one debit/credit pair,
  not two).
- [x] `FakePaymentGateway` lets a local integration test exercise both paths without
  network calls. Verified live: normal amount → `captured`; amount ending in `13` →
  `failed` with the decline reason persisted and retrievable.
- [x] Bonus, beyond the original DoD: `RefundPayment` verified live (full refund →
  `refunded` status; refunding an already-refunded payment correctly rejected as a
  conflict) and the webhook endpoint verified live (valid/invalid HMAC signature →
  `200`/`401`).
