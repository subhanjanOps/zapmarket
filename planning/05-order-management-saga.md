# Stage 5 — Order Management & Saga Orchestration

Corresponds to checklist **Phase 9** and the schema half of **Phase 12**.

## Goal

Build `order-management-service` as the saga orchestrator: own the order FSM, call
Inventory and Payment over gRPC in sequence, write the order + outbox row in one
transaction, and handle compensation when a downstream step fails. This is the
capstone stage that proves Stages 3-4 actually work together.

## Preconditions
- Stage 3 (Inventory gRPC `ReserveStock`/`ReleaseStock`) merged and runnable.
- Stage 4 (Payment gRPC `ChargeCard`) merged and runnable.
- Stage 1 migrations pattern in place.

## Tasks

### 5.1 Scaffolding
- [ ] Standard layout: `internal/domain`, `internal/repository`, `internal/service`,
      `internal/handler/http` (public checkout API), `internal/handler/grpc` (clients to
      Inventory/Payment live here or in a dedicated `internal/clients` package — prefer
      `internal/clients` since these are outbound, not inbound, gRPC handlers).
- [ ] `.env.example`: `DB_NAME=ordermanagement`, `INVENTORY_SERVICE_ADDR`,
      `PAYMENT_SERVICE_ADDR`, `AUTH_SERVICE_ADDR`, `HTTP_PORT`.

### 5.2 Domain model & FSM
- [ ] `orders` table: `id`, `user_id`, `status`, `total_amount`, `created_at`, `updated_at`.
- [ ] `order_items` table: `order_id`, `sku_id`, `quantity`, `unit_price`.
- [ ] `outbox` table (Phase 12 schema, built here since Order is the first outbox writer):
      `id`, `aggregate_type`, `aggregate_id`, `event_type`, `payload` (jsonb),
      `created_at`, `published_at` (nullable — null means Debezium/publisher hasn't
      picked it up yet).
- [ ] States per `design.md`: `PENDING -> RESERVED -> PAID -> CONFIRMED`, with
      `CANCELLED` reachable from `PENDING` and `RESERVED`. Status string values must
      be `UPPER_SNAKE_CASE` in Go constants, SQL literals, migration CHECK constraints,
      and proto/JSON fields — same convention as inventory and payment (see `00-overview.md`).
- [ ] Implement the FSM as an explicit allowed-transitions map in `internal/domain`, not
      ad-hoc if-chains in the service layer — reject illegal transitions with
      `errors.NewValidation`.

### 5.3 Checkout saga (synchronous-only for this stage)
- [ ] `POST /orders` (checkout): within a single Postgres transaction —
      1. Insert order (`PENDING`) + order_items.
      2. Call Inventory `ReserveStock` per item over gRPC. Any failure → rollback the
         transaction, return error immediately (per `design.md`: "Inventory reservation
         fails → Order returns error immediately, no payment attempted").
      3. On all reservations succeeding, transition order to `RESERVED`, commit.
      4. Call Payment `ChargeCard` over gRPC (outside the DB transaction, since it's a
         network call to a downstream service — don't hold a DB transaction open across it).
      5. On payment success: open a new transaction, set order `PAID` -> `CONFIRMED`,
         insert an `order.created`-equivalent outbox row, commit.
      6. On payment failure: call Inventory `ReleaseStock` for the reservations made in
         step 2 (compensation), set order `CANCELLED`, insert outbox row.
- [ ] `Idempotency-Key` header support per `design.md` — same Postgres-unique-constraint
      approach as Stage 4 until Stage 9's Redis cache lands.

### 5.4 Compensation correctness
- [ ] Write the compensation path test first: simulate Payment failure (use Payment's
      `FakePaymentGateway` magic-failure amount from Stage 4) and assert Inventory stock
      is restored and order ends in `CANCELLED`, not stuck in `RESERVED`.
- [ ] Handle the case where `ReleaseStock` itself fails (network blip) — this needs a
      retry-with-backoff at minimum; a full dead-letter/manual-reconciliation path is
      out of scope for this stage (flagged for Stage 12's reliability work).

### 5.5 Proto / clients
- [ ] No new public proto needed for Order itself unless another service needs to call
      it synchronously (none do, per `design.md`'s communication table). Build thin gRPC
      client wrappers in `internal/clients/inventory_client.go` and
      `internal/clients/payment_client.go` using the Stage 3/4 generated stubs from
      `pkg/proto`.

## Out of scope
- Actually publishing outbox rows to Kafka — the outbox table is written here, but the
  Debezium/publisher wiring that drains it is Stage 7. Until then, `published_at` will
  simply stay null, which is fine for testing the saga's correctness in isolation.
- Cart/session management — `design.md` mentions Redis cart locks, deferred to Stage 9.

## Definition of done
- A successful checkout reserves stock, charges payment, and lands the order in
  `CONFIRMED` with one outbox row.
- A forced payment failure releases the reserved stock and lands the order in
  `CANCELLED` with stock counts back to baseline (verified by querying Inventory after).
- Duplicate checkout requests with the same `Idempotency-Key` do not double-charge or
  double-reserve.
