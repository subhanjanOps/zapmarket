# Code Review: `order-management-service`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

One of the most architecturally sophisticated services: choreographed saga (reserve → charge → confirm/cancel), idempotency via Redis + DB, transactional outbox, RBAC-gated admin routes. Foundational decisions are correct. However: committed `.env`, a production spin-loop, missing outbox event for RESERVED transition, unbounded pagination, and silently discarded `MarkCancelled` errors on payment failure paths block production approval.

---

## Critical Findings

### CRIT-1: `.env` File With Credentials Committed
**File:** `services/order-management-service/.env`
Contains `DB_PASSWORD=zappass123`. Must be removed from git history and `.gitignore`d.

### CRIT-2: Busy-Wait Spin Loop in Production Hot-Path
**File:** `internal/service/order_service.go:118-131`
Hard `time.Sleep(50ms)` busy-poll inside HTTP goroutine, up to 10 seconds. Ignores `ctx` cancellation. At load this becomes goroutine and Redis connection exhaustion.
**Fix:** Replace with `ctx`-aware retry or return `503 Retry-After` immediately on lock contention.

### CRIT-3: Unbounded `limit` on All Paginated Queries — DoS Vector
**Files:** `internal/handler/http/order_handler.go:224-238`, `internal/repository/order_repository.go:114-124`
No upper cap on `?limit=`. Client can force millions of rows in one query.
**Fix:** Enforce `if limit > 100 { limit = 100 }` at handler layer.

### CRIT-4: `MarkReserved` Does Not Write an Outbox Event
**File:** `internal/repository/order_repository.go:157-183`
`MarkConfirmed` and `MarkCancelled` write outbox rows; `MarkReserved` does not. Violates the documented invariant that all status transitions emit an outbox event. Downstream consumers (notification-service) never receive `order.reserved`.

---

## High Priority Findings

### HIGH-1: `Checkout` Function Is 178 Lines — More Than Double the Maximum
**File:** `internal/service/order_service.go:81-258`
Five distinct concerns in one function. Extract: `resolveIdempotency`, `buildDomainItems`, `reserveInventory`, `processPayment`.

### HIGH-2: Error From `MarkCancelled` Silently Discarded on Payment Failure Paths
**File:** `internal/service/order_service.go:214-215, 223-224`
`_ = s.repo.MarkCancelled(...)` — if this fails, order is left in `RESERVED` status indefinitely with inconsistent state (stock released, payment failed, order not cancelled).

### HIGH-3: All gRPC Connections Use `insecure.NewCredentials()`
**Files:** `internal/middleware/auth.go:22`, `internal/clients/inventory_client.go:19`, `internal/clients/payment_client.go:19`
Auth tokens and payment data travel in plaintext. Must be documented at minimum; production requires mTLS.

### HIGH-4: `AdminListOrders` Uses Inline Response Struct Instead of `httpx.Paginated`
**File:** `internal/handler/http/admin_handler.go:76-93`
Creates a custom `listResp` struct inline. Rest of service uses `httpx.Paginated`. Two incompatible response shapes from the same service.

### HIGH-5: `GetBySellerID` Uses Subquery With No Covering Index
**File:** `internal/repository/order_repository.go:84-112`
`idx_order_items_seller` indexes `seller_id` but not `order_id`. PostgreSQL must fetch full row for every match.
**Fix:** `CREATE INDEX idx_order_items_seller ON order_items (seller_id, order_id)`.

### HIGH-6: Pagination Division-By-Zero Risk
**File:** `internal/handler/http/order_handler.go:184`
`page := offset/limit + 1` — fragile across handlers; needs a shared `parsePaginationPage` helper with max-limit cap.

---

## Medium Priority Findings

- **MED-1:** `GetSellerOrder` makes 2 DB calls + in-memory filter — should be a single JOIN query
- **MED-2:** `CancelOrder` releases inventory before persisting CANCELLED status — if DB write fails, stock is freed but order stays RESERVED
- **MED-3:** `compensate` uses caller's context — client disconnect cancels cleanup calls; use `context.WithoutCancel`
- **MED-4:** Outbox payloads are ad-hoc `map[string]string` — no versioning, no schema, violates event design principles
- **MED-5:** `orderService` holds concrete `*redis.Client` — violates Dependency Inversion; define a `cacheStore` interface
- **MED-6:** No validation that `seller_id` is present on checkout items — orders can be silently invisible to seller views
- **MED-7:** Swagger `@BasePath /` but routes are `/v1/...` — generates incorrect Swagger UI paths
- **MED-8:** gRPC client `ClientConn` never exposed for graceful shutdown — connection leak on restart

---

## Low Priority Findings

- **LOW-1:** `DecodeJSON` has no body size limit — `http.MaxBytesReader` needed
- **LOW-2:** `order-management-service.exe` binary committed to repo
- **LOW-3:** `OrderPaid` FSM state exists but is never written — dead state
- **LOW-4:** Some error messages use string concatenation instead of structured fields
- **LOW-5:** `KAFKA_BROKERS` missing from `.env.example` — silent relay no-op if unset
- **LOW-6:** Tests only cover `Checkout` happy path — no tests for `CancelOrder`, `GetSellerOrder`, FSM edge cases

---

## Recommended Refactoring Plan

**Sprint 1 — Pre-production blockers:**
1. Remove `.env`, `.exe` from git; rotate credentials
2. Replace spin-wait with ctx-aware retry or fail-fast 503
3. Add `MarkReserved` outbox event
4. Cap `limit` to 100 in all paginated handlers
5. Handle error from `MarkCancelled` in payment failure paths

**Sprint 2 — Correctness:**
6. Swap compensation order in `CancelOrder`: persist CANCELLED first, then release inventory
7. Use `context.WithoutCancel` in `compensate`
8. Standardize admin paginated response to `httpx.Paginated`
9. Add `Close()` to gRPC clients; wire to graceful shutdown
10. Add `http.MaxBytesReader` to `DecodeJSON`

**Sprint 3 — Architecture:**
11. Extract `Checkout` into smaller named methods
12. Define `cacheStore` interface; remove concrete `*redis.Client` from service
13. Define typed event structs for outbox payloads with version field
14. Add covering index on `order_items(seller_id, order_id)`

**Sprint 4 — Tests:**
15. Add tests for `CancelOrder`, `GetSellerOrder`, `AdminCancelOrder`, FSM edge cases
16. Add repository integration tests with testcontainers

---

## Final Verdict: CHANGES REQUIRED
