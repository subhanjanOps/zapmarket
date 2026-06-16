# Stage 4 — Payment Service

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
- [ ] Same layout convention as Stage 3: `internal/domain`, `internal/repository`,
      `internal/service`, `internal/handler/grpc`, `migrations/`, `pkg/config` wiring.
- [ ] `.env.example` with `DB_NAME=payment`, `GRPC_PORT`, and placeholder gateway
      credentials (`PAYMENT_GATEWAY_API_KEY` etc. — never commit real keys, `.env.example`
      should have empty/placeholder values only).

### 4.2 Domain model
- [ ] `transactions` table: `id`, `order_id`, `amount`, `currency`, `status`
      (`PENDING`/`SUCCEEDED`/`FAILED`/`REFUNDED`), `idempotency_key`, `gateway_ref`,
      `created_at`.
- [ ] `ledger_entries` table: double-entry rows (`debit`/`credit`, `account`, `amount`,
      `transaction_id`) per `design.md`'s "PG · Ledger" note.
- [ ] Migration `0001_init.up/down.sql`.

### 4.3 Idempotency (without Redis yet)
- [ ] Since Redis isn't on until Stage 9, enforce idempotency via a unique constraint on
      `transactions.idempotency_key` at the Postgres level for now: a duplicate charge
      request with the same key returns the existing transaction's result instead of
      double-charging. This is a correct, if slower, substitute until Stage 9 adds the
      Redis-cached fast path.

### 4.4 Proto contract
- [ ] `pkg/proto/payment/v1/payment.proto`:
      `ChargeCard(order_id, amount, currency, idempotency_key) -> (transaction_id, status)`,
      `RefundPayment(transaction_id) -> (status)`, `GetTransaction(transaction_id)`.
- [ ] Generate stubs into `pkg/proto/payment`.

### 4.5 Gateway integration
- [ ] Integrate one real gateway SDK behind an interface (`internal/domain/contracts.PaymentGateway`)
      so the concrete Stripe/Razorpay client is swappable and mockable in tests. Pick
      whichever gateway `design.md`'s "Razorpay / Stripe" note implies the team prefers —
      ask the user if unspecified, don't guess silently since this affects which SDK and
      webhook signature scheme gets wired in.
- [ ] For local/dev, a `FakePaymentGateway` that always succeeds (or succeeds/fails based
      on a magic amount, e.g. amount ending in `.13` fails) so the saga can be tested
      end-to-end without a live gateway account.

### 4.6 Webhook handling
- [ ] HTTP endpoint (`internal/handler/http`) for the gateway's async webhook
      (payment confirmed/failed out-of-band) — verify signature, update transaction status.

## Out of scope
- Kafka publishing of `payment.processed` / `payment.failed` — Stage 7.
- Redis idempotency cache — Stage 9 (Postgres unique constraint covers correctness until then).
- Refund saga / compensation flows beyond a basic `RefundPayment` RPC — full refund
  workflow UI/process is not in the current checklist scope.

## Definition of done
- `go run ./services/payment-service/...` starts a gRPC server.
- Two identical `ChargeCard` calls with the same `idempotency_key` return the same
  `transaction_id` and do not create two ledger entries.
- `FakePaymentGateway` lets a local integration test exercise success and failure paths
  without network calls.
