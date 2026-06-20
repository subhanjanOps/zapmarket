# Code Review: `inventory-service`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

One of the most architecturally complete services: Redis Lua atomic check-and-decrement, PostgreSQL transactional durability, outbox relay, clean domain modeling, and meaningful test coverage for `ReserveStock`. However: committed `.env`, silent outbox event loss, TOCTOU race in `ReleaseStock`, no gRPC auth, and Redis counter drift after `AddStock` failure block production deployment.

---

## Critical Findings

### C-1: `.env` File Committed With Real Credentials
**File:** `services/inventory-service/.env`
Contains `DB_PASSWORD=zappass123`, `REDIS_URL`, `KAFKA_BROKERS`. Must be removed from git history and `.gitignore`d.

### C-2: Silent JSON Marshal Error in Outbox Path — Events Silently Dropped
**File:** `internal/repository/inventory_repository.go:106, 154`
```go
payload, _ := json.Marshal(...)
```
Error silently discarded. If marshal fails, `payload` is nil → `NULL` JSONB inserted → outbox relay publishes malformed/empty event. Violates "Never ignore errors."

### C-3: `ReleaseStock` Has TOCTOU Race — Can Double-Free Stock in Redis
**File:** `internal/service/inventory_service.go:168-179`
`GetReservationDetails` and `ReleaseStock` are two separate DB round-trips. The correct fix: return `(skuID, qty)` directly from `repo.ReleaseStock` to eliminate the gap and the extra round-trip.

### C-4: No Authentication or Authorization on Any gRPC Endpoint
**File:** `internal/handler/grpc/inventory_grpc_handler.go`, `main.go:87`
`AddStock` is a privileged write operation completely open to unauthenticated callers. No JWT validation middleware.

---

## High Priority Findings

### H-1: Redis Counter Becomes Permanently Stale After `AddStock` + Redis Failure
**File:** `internal/service/inventory_service.go:71-73`
If Redis `INCRBY` fails during `AddStock`, the warning is logged and execution continues. Redis now holds a lower quantity than Postgres indefinitely — future `ReserveStock` calls produce false `INSUFFICIENT_STOCK` errors.
**Fix:** `DEL` the key on Redis write failure, letting the cache-miss path re-warm it.

### H-2: `ReserveStock` gRPC Handler Has Dead Code and Misleading Semantics
**File:** `internal/handler/grpc/inventory_grpc_handler.go:51-55`
`if reservation == nil` check is dead — service never returns `(nil, nil)`. The interface comment in `contracts/repositories.go` says it does. Pick one contract and enforce it consistently.

### H-3: `inventory-service.exe` Binary Committed to Repository
**File:** `services/inventory-service/inventory-service.exe`
Compiled binary in source control. Delete and add `*.exe` to `.gitignore`.

### H-4: Interface Contract Ambiguity — Nil Reservation vs. Error for Insufficient Stock
**File:** `internal/domain/contracts/repositories.go:26-29`
Contract doc says "Returns (nil, nil) if insufficient stock — not an error." Service wraps nil into `pkgerrors.NewConflict("INSUFFICIENT_STOCK")`. Ambiguous contract across all callers.

### H-5: `AddStock` Ledger Entry Uses Confusing `qtyBefore`/`qtyAfter` Naming
**File:** `internal/repository/inventory_repository.go:37-46`
Variables track `qty_on_hand` in some operations and `qty_reserved` in others, making the ledger semantics inconsistent. Not a bug but a maintainability hazard.

---

## Medium Priority Findings

- **M-1:** `InventoryService` interface defined in `service` package — should be in `domain/contracts/` per DIP
- **M-2:** Comment in `dbReserve` fallback says "not atomic" — incorrect; the DB path IS atomic (single UPDATE statement in transaction)
- **M-3:** `GetReservationDetails` read has no transaction isolation — returns stale data for Redis increment
- **M-4:** No expiry sweep for `RESERVED` reservations past `expires_at` — abandoned reservations lock stock permanently
- **M-5:** Health endpoint always returns `{"status":"ok"}` — doesn't probe DB or Redis for Kubernetes readiness
- **M-6:** `.env.example` missing `REDIS_URL`, `KAFKA_BROKERS`, `MIGRATE_ON_BOOT` — developer setup broken out of box
- **M-7:** `DeductStock` writes no outbox event — `inventory.deducted` is a core business fact missing from the event stream

---

## Low Priority Findings

- **L-1:** `DefaultWarehouseID` declared as `var` — any code can reassign it; should be unexported with getter
- **L-2:** `NewInventoryRepository` returns concrete type — should return `contracts.InventoryRepository`
- **L-3:** `strconv.Itoa` for Lua argv — minor style inconsistency
- **L-4:** `GetStock` has no structured log call — inconsistent observability vs. other service methods
- **L-5:** `go.mod` declares `go 1.25.0` — Go 1.25 does not exist; causes toolchain mismatch

---

## Recommended Refactoring Plan

**Priority 1 — Do Before Merging:**
1. Remove `.env`, `.exe` from git; rotate credentials; add to `.gitignore`
2. Fix silent `json.Marshal` error in outbox writes
3. Add gRPC auth interceptor to `grpcx.NewServer()` for `AddStock`
4. Update `.env.example` with all required vars

**Priority 2 — Next Sprint:**
5. Refactor `ReleaseStock`: return `(skuID, qty)` from repo directly, eliminate TOCTOU gap
6. Fix Redis staleness after `AddStock` failure: `DEL` key on Redis write error
7. Add `inventory.deducted` outbox event to `DeductStock`
8. Resolve `InventoryService` contract ambiguity (nil vs. error for insufficient stock)
9. Move `InventoryService` interface to `internal/domain/contracts/`

**Priority 3 — Tech Debt:**
10. Add `/ready` endpoint with DB and Redis ping
11. Make `DefaultWarehouseID` unexported
12. Change `NewInventoryRepository` to return interface type
13. Fix `go 1.25.0` in `go.mod`
14. Add integration tests for `AddStock`, `ReleaseStock`, `DeductStock` service paths
15. Open a tracked issue for the expiry sweep on abandoned reservations

---

## Final Verdict: CHANGES REQUIRED
