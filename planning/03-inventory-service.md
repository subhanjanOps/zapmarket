# Stage 3 — Inventory Service

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
- [ ] Restructure `services/inventory-service` to match the established layout:
      `cmd/main.go` (or root `main.go`, matching whichever convention Stage 1 settled on),
      `internal/domain`, `internal/repository`, `internal/service`, `internal/handler/grpc`,
      `internal/handler/http` (for ops/health endpoints only — no public REST surface per
      `design.md`), `pkg/config` wiring, `migrations/`.
- [ ] Add `.env.example` with `DB_NAME=inventory` (or `inventorysvc` to avoid clashing with
      the word "inventory" as a SQL keyword-adjacent name — verify against `db-design.md`),
      `GRPC_PORT`, `AUTH_SERVICE_ADDR` if any endpoint needs auth.

### 3.2 Domain model
- [ ] `inventory_ledger` table: `sku_id`, `warehouse_id` (or single-warehouse for now if
      `db-design.md` doesn't call for multi-warehouse), `quantity_available`,
      `quantity_reserved`, `updated_at`. Check `db-design.md` for the authoritative schema
      before inventing one.
- [ ] Migration `0001_init.up/down.sql` under `migrations/`.

### 3.3 Proto contract
- [ ] Create `pkg/proto/inventory/v1/inventory.proto` with RPCs:
      `ReserveStock(sku_id, quantity, order_id) -> (reserved: bool, reservation_id)`,
      `ReleaseStock(reservation_id)`, `DeductStock(reservation_id)` (confirms the reserved
      stock as permanently deducted after payment succeeds), `GetStock(sku_id) -> quantity`.
- [ ] Generate Go stubs via `protoc --go_out=. --go-grpc_out=.` per the centralized
      `pkg/proto` pattern already used by `auth` and `catalog`.

### 3.4 Service logic
- [ ] `ReserveStock`: atomic check-and-decrement at the Postgres layer for now (a single
      `UPDATE ... WHERE quantity_available >= $qty RETURNING ...` guarded query) — the
      Redis Lua-script version is deferred to Stage 9 once Redis is turned on; don't block
      this stage on Redis.
  - Why Postgres-only first: Stage 9 explicitly turns Redis on platform-wide; building a
    fast-path here now means rewriting it then anyway, but Order/Payment need a *working*
    Inventory now to unblock Stage 5.
- [ ] `ReleaseStock` / `DeductStock` transition the reservation row's state
      (`PENDING -> RELEASED` / `PENDING -> DEDUCTED`).
- [ ] Reservations should have a TTL/expiry concept (even if enforced by a cron-ish sweep
      for now) so abandoned carts don't permanently lock stock.

### 3.5 gRPC server
- [ ] Implement the server using `pkg/grpcx` bootstrap helpers (recovery interceptor,
      logging interceptor) — same pattern as `auth-service`'s gRPC server.

## Out of scope
- Kafka publishing of `inventory.reserved` / `inventory.released` — stubbed as TODO until
  Stage 7 turns Kafka on; until then, Order Management (Stage 5) talks to Inventory via
  gRPC only, synchronously.
- Redis-backed atomic counters and distributed locks — Stage 9.

## Definition of done
- `go run ./services/inventory-service/...` starts a gRPC server on the configured port.
- A manual `grpcurl` (or a small Go test client) can call `ReserveStock` and see
  `quantity_available` decrement in Postgres, and `ReleaseStock` restore it.
- Concurrent reservation requests for the same SKU never oversell (verify with a quick
  concurrent-request test — e.g. 20 goroutines reserving from a stock of 10, exactly 10
  succeed).
