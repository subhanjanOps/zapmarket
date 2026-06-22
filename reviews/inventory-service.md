# inventory-service Review

**Reviewer:** Principal Engineer
**Date:** 2026-06-22
**Branch:** features/cluster-setup

## Executive Summary

The inventory-service is the most technically sophisticated of the three reviewed services. It implements a Redis Lua script check-and-decrement for atomic, oversell-safe stock reservation, with a DB-only fallback path when Redis is unavailable, a Redis counter rollback on Postgres rejection, a full stock movement ledger, reservation TTL tracking, double-publish prevention for release/deduct via status-checking SQL, and a transactional outbox on every mutation. It exposes a gRPC interface (ReserveStock, ReleaseStock, DeductStock, AddStock, GetStock) and a health-check HTTP endpoint. It is the only service of the three with a test file covering the critical Redis/DB interaction paths. The primary gaps are: no test for `AddStock`/`ReleaseStock`/`DeductStock` paths, a missing expired-reservation sweep job, a Redis counter that can permanently drift below the true DB value under specific race conditions, and no auth on the gRPC server.

---

## Critical Findings

### C1 — Redis counter can permanently drift negative under a DB-reject-then-Redis-rollback-failure race
**File:** `internal/service/inventory_service.go` lines 133-151
After the Lua script decrements the Redis counter, if the Postgres `ReserveStock` fails, the service attempts to restore the counter with `rdb.IncrBy(ctx, key, int64(qty))`. If this `IncrBy` also fails (Redis transient error), the counter is permanently lower than the true `qty_available` in Postgres. The service logs a CRITICAL message but takes no further action. The effect is that future `ReserveStock` calls will return `INSUFFICIENT_STOCK` for orders that Postgres could actually fulfill — stock is effectively locked away until the Redis key is deleted and re-warmed.
**Fix:** Implement a periodic reconciliation job (or a recovery path triggered on `CRITICAL` log) that compares the Redis counter against `SELECT qty_available FROM inventory WHERE sku_id = $1` and resets the key if they diverge. Alternatively, set a short TTL (e.g., 5 minutes) on all stock keys so stale counters self-heal, and always fall through to the DB path on a cache miss after TTL expiry.

### C2 — No expired-reservation sweep job
**File:** `internal/domain/models.go` line 80, `internal/repository/inventory_repository.go` `ReserveStock`
`ReservationTTL = 15 * time.Minute` and `ExpiresAt` is set on every reservation row. The domain comment explicitly acknowledges that no sweep job exists. In production, if order-management-service fails to call `ReleaseStock` after a checkout failure (crash, network partition), inventory remains reserved forever. The `qty_reserved` column never decrements, and `qty_available` converges to zero over time even though real stock exists.
**Fix:** Implement a background goroutine (or a cron job as a separate task) that runs every minute and executes:
```sql
UPDATE inventory i
SET qty_reserved = qty_reserved - r.qty, updated_at = NOW()
FROM inventory_reservations r
WHERE r.inventory_id = i.id
  AND r.status = 'RESERVED'
  AND r.expires_at < NOW()
RETURNING r.id, r.sku_id, r.qty;
```
Then update those reservations to `RELEASED`, write outbox events, and increment the Redis counters.

---

## High Priority Findings

### H1 — gRPC server has no authentication or authorization
**File:** `internal/handler/grpc/inventory_grpc_handler.go`, `main.go`
The gRPC server accepts `AddStock`, `ReserveStock`, `ReleaseStock`, and `DeductStock` from any in-cluster caller without verifying identity. A rogue service or a misconfigured client could add arbitrary stock or release reservations that belong to a different order.
**Fix:** Add a `UnaryServerInterceptor` that enforces mTLS or verifies a shared service-to-service secret. `AddStock` in particular (which increases `qty_on_hand`) should require an `admin` or `warehouse` role claim, not just any service credential.

### H2 — `AddStock` does not validate that the calling service is authorized to add to this SKU
Even if H1's auth interceptor is added, `AddStock` should verify that the SKU exists in product-catalog-service before accepting stock. Currently any caller can add stock for a non-existent SKU ID, creating orphaned inventory rows.
**File:** `internal/service/inventory_service.go` lines 64-86
**Fix:** Before calling `s.repo.AddStock`, validate the SKU ID against product-catalog-service via gRPC. Accept an optional `productCatalogClient` dependency in `inventoryService`.

### H3 — `dbReserve` fallback comment says "not atomic" but the SQL is actually atomic (single row update with WHERE guard)
**File:** `internal/service/inventory_service.go` lines 157-168
The comment says: "full check-and-decrement in Postgres (original single-DB behaviour, correct but not atomic)". The Postgres `UPDATE ... WHERE qty_on_hand - qty_reserved >= $2 ... RETURNING` is actually a single atomic statement under Postgres's row-level locking — it is as atomic as the Lua script. The misleading comment could cause future maintainers to distrust the fallback and introduce unnecessary locking mechanisms.
**Fix:** Update the comment: "fallback to Postgres path — single-statement atomic check-and-update via row lock; no Redis counter update since Redis is unavailable".

### H4 — Redis counter is never given a TTL on warm/update
**File:** `internal/service/inventory_service.go` lines 116, 137
`s.rdb.Set(ctx, key, inv.QtyAvailable, 0)` passes TTL `0` (no expiry). If the Postgres and Redis values diverge (DB migration, manual correction, C1 scenario), the Redis counter will permanently override the DB until the process restarts or the key is manually deleted. This compounds the drift problem in C1.
**Fix:** Set a bounded TTL (e.g., `5 * time.Minute`) on the cache key. A cache miss after expiry re-warms from DB, self-healing any drift. This does not remove the need for the reconciliation job in C1 but provides a safety net.

---

## Medium Priority Findings

### M1 — Incomplete test coverage for `AddStock`, `ReleaseStock`, and `DeductStock`
**File:** `internal/service/inventory_service_test.go`
The test file covers `ReserveStock` thoroughly (Redis hit, cache miss, insufficient stock, Redis down, DB rejection rollback, validation). `AddStock`, `ReleaseStock`, `DeductStock`, and `GetStock` have zero test cases.
**Fix:** Add tests for:
- `AddStock`: Redis key incremented when warm; Redis key not created when absent (the `luaIncrIfExists` path).
- `ReleaseStock`: Redis counter incremented after DB release; error from DB propagated.
- `DeductStock`: DB `CONFIRMED` transition; Redis counter not changed (correct behavior per comment).
- `GetStock`: returns domain struct correctly.

### M2 — Outbox event for `inventory.released` uses `reservationID` as `aggregate_id`, but `inventory.reserved` uses `reservation.ID`
**File:** `internal/repository/inventory_repository.go` lines 163-166 vs lines 107-114
Both use the reservation UUID as `aggregate_id`, which is correct and consistent. This is fine. However, `DeductStock` writes no outbox event at all — there is no `inventory.deducted` or `inventory.confirmed` event. Downstream consumers (reporting, notification) cannot observe when stock is permanently consumed.
**Fix:** Add `insertOutboxRow(ctx, tx, reservationID, "inventory", "inventory.deducted", payload)` inside the `DeductStock` transaction.

### M3 — `GetReservationDetails` is defined in the interface and implemented in the repository but never called by the service
**File:** `internal/domain/contracts/repositories.go` line 52, `internal/repository/inventory_repository.go` lines 248-261
`GetReservationDetails` was presumably added for the `ReleaseStock` path to look up `skuID` without a round-trip, but `ReleaseStock` in the repository already returns `skuID` via `RETURNING`. The contract method is dead code.
**Fix:** Remove `GetReservationDetails` from the `InventoryRepository` interface and its implementation. Dead interface methods violate the Interface Segregation Principle.

### M4 — `DefaultWarehouseID` is a package-level `var`, not a `const`
**File:** `internal/domain/models.go` line 13
```go
var DefaultWarehouseID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
```
`uuid.UUID` is a `[16]byte` array — it cannot be a `const`, so `var` is technically required. However, nothing prevents mutation of this value. Rename to `defaultWarehouseID` (unexported) and expose it via a function `DefaultWarehouse() uuid.UUID` to prevent accidental reassignment, or document that it is immutable.

### M5 — No distributed tracing on gRPC server
Per engineering standards, all services must support distributed tracing. The gRPC server has no OpenTelemetry `UnaryServerInterceptor`. Calls from order-management-service carry no trace context.
**Fix:** Add `otelgrpc.UnaryServerInterceptor()` to `grpcx.NewServer()` or inject in `main.go`.

### M6 — `luaReserve` Lua script has no protection against negative counters
**File:** `internal/service/inventory_service.go` lines 32-39
```lua
return redis.call('DECRBY', KEYS[1], qty)
```
The script checks `available < qty` and returns 0 (insufficient). However, if two concurrent requests both see `available = 5` and one decrements to 2 while the other concurrently passes the check (due to a non-atomic read in a non-Lua context — impossible here, but worth noting), the counter could go negative. In the current single-script implementation this is safe because the `GET` and `DECRBY` are atomic within the Lua script. The existing behavior is correct, but a guard `if available - qty < 0 then return 0 end` would make the invariant explicit and protect against future script modifications.

---

## Low Priority Findings

### L1 — `MovementType` constants use `snake_case` strings instead of `UPPER_SNAKE_CASE`
**File:** `internal/domain/models.go` lines 46-52
```go
MovementPurchaseOrder      MovementType = "purchase_order"
MovementReservation        MovementType = "reservation"
```
All other status enums in the project use UPPER_SNAKE_CASE per the project memory standard. Ledger movement types should follow suit.
**Fix:** Change to `"PURCHASE_ORDER"`, `"RESERVATION"`, etc. Add a migration to update existing rows.

### L2 — `ReservationStatus` constants are UPPER_SNAKE_CASE but `MovementType` is not (inconsistency within the same file)
Related to L1. `ReservationReserved = "RESERVED"`, `ReservationConfirmed = "CONFIRMED"`, `ReservationReleased = "RELEASED"` are correct. The `MovementType` constants in the same file should match.

### L3 — Health handler is a package-level function, not a method — inconsistent with other services
**File:** `internal/handler/http/health_handler.go`
`httphandler.Health` is registered as a bare function. All other HTTP handlers in the codebase are methods on handler structs. This is a minor inconsistency that makes future extension (e.g., adding a DB ping check to the health response) awkward.
**Fix:** Wrap in a `HealthHandler` struct with a `Health` method, even if the struct has no fields currently.

### L4 — `inventory_service_test.go` uses `slog.Default()` directly instead of a test-scoped logger
Minor: Using `slog.Default()` in tests means test output is mixed with application logs. Pass `slog.New(slog.NewTextHandler(io.Discard, nil))` for silent tests.

---

## Recommended Refactoring Plan

**Sprint 1 (before any production traffic):**
1. Fix C2: Implement expired-reservation sweep goroutine in `main.go`.
2. Fix H1: Add gRPC `UnaryServerInterceptor` for service-to-service auth.
3. Fix H4: Set a 5-minute TTL on all Redis stock keys (self-healing for drift).

**Sprint 2:**
4. Fix C1: Implement Redis/DB reconciliation on CRITICAL log or as a scheduled job.
5. Fix M1: Add unit tests for `AddStock`, `ReleaseStock`, `DeductStock`, `GetStock`.
6. Fix M2: Add `inventory.deducted` outbox event in `DeductStock`.
7. Fix M3: Remove `GetReservationDetails` from the interface (dead code).

**Sprint 3:**
8. Fix H2: Validate SKU existence against product-catalog-service in `AddStock`.
9. Fix L1/L2: Standardize `MovementType` to UPPER_SNAKE_CASE with migration.
10. Add OpenTelemetry server interceptor (M5).
11. Fix H3: Update misleading `dbReserve` comment.

---

## Final Verdict

**Approved with required changes.** The inventory-service demonstrates the highest level of engineering among the three reviewed services. The Lua check-and-decrement pattern is correctly implemented, the Redis fallback and rollback logic is sound, and the test suite for `ReserveStock` is exemplary. Two critical production issues — the absence of an expired-reservation sweep job and the potential for permanent Redis counter drift — must be resolved before this service handles real orders at scale. The missing gRPC auth (H1) is required before any non-order-management-service caller can be permitted. All other findings are incremental hardening items that can be addressed in subsequent sprints.
