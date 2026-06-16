# Stage 3 — Inventory Service

**Status: ✅ Complete (2026-06-16)**

Corresponds to checklist **Phase 8**.

## Goal

Build `inventory-service` from its current scaffold (`go.mod` + placeholder `main.go`)
into a working service that tracks stock per SKU/warehouse and exposes gRPC reserve/
release/deduct operations — the first dependency Order Management needs for its saga.

## Preconditions
- Stage 1 (domain contracts pattern, migrations) merged — Inventory should be built
  using that pattern from the start, not retrofitted later.
- Stage 2 not strictly required but gives a sense of the SKU shape this service tracks
  against (`sku_id` from product-catalog-service).

## Tasks

### 3.1 Scaffolding
- [x] Built with root `main.go` (matching `product-catalog-service`'s convention, not
      `cmd/main.go`): `internal/domain`, `internal/domain/contracts`, `internal/repository`,
      `internal/service`, `internal/handler/grpc`, `internal/handler/http` (health-check
      only — no public REST surface, per `design.md`), `migrations/`.
- [x] `.env.example` with `DB_NAME=inventory` (matches the DB already created by
      `docker-entrypoint-initdb.d/init.sql`), `HTTP_PORT=8082`, `GRPC_PORT=50053`
      (next free ports after auth's 8080/50051 and catalog's 8081/50052). No
      `AUTH_SERVICE_ADDR` — none of the RPCs need end-user auth (Inventory is only ever
      called service-to-service, by Order Management).

### 3.2 Domain model
- [x] Used the **authoritative** schema already in `db-design.md` §3 (and previously
      duplicated in `docker-entrypoint-initdb.d/init.sql`) rather than inventing a
      simplified one: `warehouses`, `inventory` (per-SKU-per-warehouse stock with a
      generated `qty_available` column), `inventory_ledger` (immutable movement audit
      trail), `inventory_reservations` (TTL-bearing, tied to an order).
- [x] Migration `0001_init.up/down.sql` — also seeds a single `Default Warehouse` row
      (fixed UUID `00000000-...-000000000001`), since the proto surface doesn't expose
      `warehouse_id` yet (single-warehouse for this stage, per the original plan).
- [x] Trimmed the now-redundant copy of this schema out of
      `docker-entrypoint-initdb.d/init.sql`, same pattern as Stage 1 did for
      `auth-service`/`product-catalog-service`.

### 3.3 Proto contract
- [x] `pkg/proto/inventory/inventory.proto` with `AddStock`, `ReserveStock`,
      `ReleaseStock`, `DeductStock`, `GetStock`. **Added `AddStock` beyond the original
      plan** — without it there's no way to ever get stock into the system at all
      (checklist Phase 8 lists "Inventory CRUD" as in-scope; this is the minimal
      create-side of that).
- [x] Generated via `protoc --go_out=. --go-grpc_out=.` — note: protoc's `go_package`
      resolution under plain `--go_out=.` (no `module=...` flag) writes output to a
      nested `github.com/...` path tree; had to move the generated files into
      `pkg/proto/inventory/` manually, matching whatever the `auth`/`catalog` packages'
      generation must have also done by hand.

### 3.4 Service logic
- [x] `ReserveStock`: single guarded `UPDATE ... WHERE qty_on_hand - qty_reserved >= $qty
      RETURNING ...` inside a DB transaction — Postgres's row-level lock on the `UPDATE`
      serializes concurrent reservations against the same SKU correctly. Redis Lua-script
      fast path still deferred to Stage 9 as planned.
- [x] `ReleaseStock` / `DeductStock` transition the reservation row's state using the
      schema's actual enum (`reserved -> released` / `reserved -> confirmed`), not the
      `PENDING`/`DEDUCTED` names the original plan sketch used — aligned to
      `db-design.md`'s real `CHECK` constraint instead of inventing new state names.
      Both reject an already-non-`reserved` row with a `409`-equivalent conflict
      (`RESERVATION_NOT_RESERVED`) rather than silently double-applying the effect.
- [x] `inventory_reservations.expires_at` is set (`domain.ReservationTTL` = 15 minutes)
      on every reservation. The actual sweep job that auto-releases expired rows is
      **not built** — out of scope per the original plan ("even if enforced by a cron-ish
      sweep for now" was aspirational; no sweep exists yet). Flagging this as a real gap:
      an abandoned cart's stock stays locked for 15 minutes with nothing currently
      enforcing the expiry. Revisit when Order Management (Stage 5) exists to decide
      whether the sweep lives here or there.

### 3.5 gRPC server
- [x] Built on `pkg/grpcx.NewServer()` (recovery + logging interceptors), identical
      pattern to `auth-service` and `product-catalog-service`.

## Out of scope
- Kafka publishing of `inventory.reserved` / `inventory.released` — stubbed as TODO until
  Stage 7 turns Kafka on; until then, Order Management (Stage 5) talks to Inventory via
  gRPC only, synchronously.
- Redis-backed atomic counters and distributed locks — Stage 9.

## Definition of done
- [x] Service starts a gRPC server on the configured port. Verified via Docker
  (`docker compose up inventory-service` → `healthy`, logs show `starting gRPC server
  port=50053`).
- [x] Manual `grpcurl` calls verified the full lifecycle live: `AddStock` 50 →
  `GetStock` shows 50 available → `ReserveStock` 10 → available drops to 40 →
  `ReleaseStock` → available back to 50. Also verified `DeductStock`: reserved 5,
  deducted → `qty_on_hand` permanently dropped 50→45, `qty_reserved` back to 0. Also
  verified double-release is rejected (`AlreadyExists` / `RESERVATION_NOT_RESERVED`),
  not silently double-applied.
- [x] Concurrency test: seeded stock of 10, fired 20 concurrent `ReserveStock` requests
  (qty 1 each) via parallel `grpcurl` processes. Exactly 10 succeeded
  (`reserved: true`), 10 got `reserved: false` (not errors — the expected
  "insufficient stock" outcome). Final state: `qty_on_hand=10, qty_reserved=10,
  qty_available=0` — no oversell.
