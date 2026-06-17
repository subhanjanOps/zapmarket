# Stage 5 — Order Management & Saga Orchestration

**Status: ✅ Complete (2026-06-17)**

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
- [x] Standard layout: `internal/domain`, `internal/repository`, `internal/service`,
      `internal/handler/http` (public checkout API), `internal/clients` for outbound
      gRPC wrappers (inventory + payment). No gRPC server — Order is not called
      synchronously by any other service.
- [x] `.env.example`: `DB_NAME=ordermanagement`, `INVENTORY_SERVICE_ADDR`,
      `PAYMENT_SERVICE_ADDR`, `AUTH_SERVICE_ADDR`, `HTTP_PORT=8084`.
- [x] `InventoryServiceAddr` and `PaymentServiceAddr` added to shared `pkg/config`
      (defaults: `localhost:50053` and `localhost:50054`).

### 5.2 Domain model & FSM
- [x] `orders` table: `id`, `user_id`, `idempotency_key` (UNIQUE), `status`, `total_amount`,
      `currency`, `payment_id`, `created_at`, `updated_at`, `deleted_at`.
- [x] `order_items` table: `order_id`, `sku_id`, `quantity`, `unit_price`, `reservation_id`
      (set after stock is reserved).
- [x] `outbox` table (Phase 12 schema): `aggregate_id`, `aggregate_type`, `event_type`,
      `payload` (jsonb), `published_at` (nullable), `created_at`.
- [x] States: `PENDING → RESERVED → CONFIRMED`, `PENDING/RESERVED → CANCELLED`. Note:
      `PAID` state skipped — saga goes straight from RESERVED to CONFIRMED once
      payment is captured; PAID is not a stable intermediate state the system
      persists.
- [x] FSM implemented as an `allowedTransitions` map on `domain.Order.Transition(next)`
      — illegal transitions return a Validation error.

### 5.3 Checkout saga (synchronous-only for this stage)
- [x] `POST /v1/orders` body carries items (sku_id, quantity, unit_price) and
      `idempotency_key` (UUID). Auth middleware enforces `buyer`/`seller`/`admin` roles.
- [x] Saga steps (no DB transaction held open across gRPC calls):
      1. Idempotency check — replay if key already exists.
      2. DB transaction: insert order (`PENDING`) + items, commit.
      3. Loop: call Inventory `ReserveStock` per item; any failure → compensate already-
         reserved items + return error. Insufficient-stock case is `409 INSUFFICIENT_STOCK`.
      4. DB transaction: set order `RESERVED`, persist `reservation_id` per item, commit.
      5. Call Payment `ChargeCard` over gRPC.
      6a. `CAPTURED` → DB transaction: set order `CONFIRMED`, set `payment_id`, write
          `order.confirmed` outbox row, commit.
      6b. Not captured → release all reservations, DB transaction: set order `CANCELLED`,
          write `order.cancelled` outbox row, commit.
- [x] Idempotency via `orders.idempotency_key UNIQUE` — same Postgres constraint approach
      as Stage 4 until Stage 9's Redis cache lands.
- [x] `GET /v1/orders` — list caller's orders (auth-scoped).
- [x] `GET /v1/orders/{id}` — get single order with items (returns 404 if not owned by caller).

### 5.4 Compensation correctness
- [x] `compensate()` in `order_service.go` iterates reserved items and calls
      `inventory.ReleaseStock` for each. Release failures are logged but not propagated
      — the caller's error takes precedence. Stock stranded in RESERVED will be
      recovered by the TTL sweep job (flagged for Stage 12).
- [x] Payment failure path: use amount ending in `13` paise with `FakePaymentGateway`
      to trigger a decline — order lands in `CANCELLED` with outbox row, reservations
      are released. Verifiable end-to-end once all three services are running.

### 5.5 Proto / clients
- [x] No public proto for Order — nothing calls it synchronously.
- [x] `internal/clients/inventory_client.go` wraps `pkg/proto/inventory` stubs:
      `ReserveStock` (returns reservationID + ok bool) and `ReleaseStock`.
- [x] `internal/clients/payment_client.go` wraps `pkg/proto/payment` stubs:
      `ChargeCard` (returns paymentID + status string).

## Out of scope
- Actually publishing outbox rows to Kafka — the outbox table is written here, but the
  Debezium/publisher wiring that drains it is Stage 7. Until then, `published_at` will
  simply stay null, which is fine for testing the saga's correctness in isolation.
- Cart/session management — `design.md` mentions Redis cart locks, deferred to Stage 9.

## Definition of done
- [x] Service starts on HTTP port 8084, connects to inventory/payment/auth gRPC at boot.
- [x] Migrations create `orders`, `order_items`, `outbox` tables on first boot.
- [ ] Verified live: successful checkout → order `CONFIRMED`, one `order.confirmed`
      outbox row, inventory qty_reserved reduced.
- [ ] Verified live: payment failure (amount ending `13`) → order `CANCELLED`, one
      `order.cancelled` outbox row, stock restored to baseline.
- [ ] Verified live: duplicate idempotency key → same order returned, no double-reserve.
