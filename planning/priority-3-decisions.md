# Priority 3 — Decision Report

**Branch:** `features/cluster-setup`  
**Date:** 2026-06-19  
**Author:** Code Review (Principal Engineer)

---

## Overview

Priority 3 is a quality-and-maintainability sweep: no new features, no new APIs. The five items are interdependent in one way — item 9 (shared relay package) is a prerequisite for item 13 (tests) because tests should import the canonical implementation, not one of the three copies. Everything else is independent.

---

## Item 9 — Move `OutboxRelay` to `pkg/relay/`

### Problem

The identical `OutboxRelay` implementation exists verbatim in:
- `services/order-management-service/internal/relay/outbox_relay.go`
- `services/payment-service/internal/relay/outbox_relay.go`
- `services/inventory-service/internal/relay/outbox_relay.go`

This was proven dangerous in Priority 1: the `logger` assignment bug had to be fixed in three separate files. Any future change to relay behaviour (batch size, retry policy, dead-letter handling) must be applied three times. The three files share zero differentiating logic — they are true copies, not variants.

### Alternatives Considered

| Approach | Dedup | Module isolation | Effort |
|---|---|---|---|
| Keep three copies (current) | ❌ | ✅ | None |
| Shared internal package inside one service | ❌ can't import across service modules | — | — |
| `pkg/relay/` as a new Go module in the workspace | ✅ | ✅ | Medium |
| Generic package inside `pkg/kafka/` | ✅ | ✅ coupling concern | Low |

### Decision

**Create `pkg/relay/` as a new workspace module.** This follows the established pattern — `pkg/kafka`, `pkg/redis`, `pkg/migrate` are all standalone modules in `go.work`. The relay package has two dependencies: `database/sql` (stdlib) and `pkg/kafka`. It takes a `*sql.DB`, a `*pkgkafka.Producer`, a topic string, and a `*slog.Logger`. The three service relay packages are deleted; each service's `main.go` is updated to import `pkg/relay` instead.

The three internal `relay/` directories in the service trees are removed entirely.

### Interface shape

```go
// pkg/relay/relay.go
func New(db *sql.DB, producer *pkgkafka.Producer, topic string, logger *slog.Logger) *OutboxRelay
func (r *OutboxRelay) Run(ctx context.Context)
```

No behavioural change — identical to the current implementation, now in one place.

---

## Item 10 — Add Pagination to `ListOrders` and `ListSellerOrders`

### Problem

Both endpoints return every order for the caller with no limit:

```go
// service layer
func (s *orderService) ListOrders(ctx context.Context, userID uuid.UUID) ([]*domain.Order, error) {
    return s.repo.GetByUserID(ctx, userID)  // no LIMIT/OFFSET
}
```

The repository queries use `ORDER BY created_at DESC` with no `LIMIT`. A buyer with 500 orders, or a marketplace seller with 10,000 orders, causes a full table scan and returns the entire result set to the handler. The admin `ListAll` method already has `OrderListParams{Limit, Offset}` — the pattern to follow exists.

### What Changes

**New type in `contracts`:**
```go
type OrderPageParams struct {
    Limit  int
    Offset int
}
```

**Repository:** `GetByUserID` and `GetBySellerID` accept `OrderPageParams` and append `LIMIT $N OFFSET $M` to their queries. They also return a `total int64` (via a preceding `COUNT(*)`) so the HTTP response can include pagination metadata.

**Service interface:** `ListOrders` and `ListSellerOrders` accept `limit, offset int` and return `([]*domain.Order, int64, error)`.

**Handler:** Reads `?limit=` and `?offset=` query params (defaulting to 20 and 0); returns a paginated envelope using the existing `httpx.Paginated` helper (already used by `product-catalog-service`).

**Swagger docs:** Updated for both endpoints.

### Why not reuse `OrderListParams`?

`OrderListParams` also carries `Status`, `UserID`, `From`, `To` which are admin-only filters. Mixing admin filter fields into the buyer/seller list params leaks admin-surface into user-facing contracts. A separate `OrderPageParams` is minimal and explicit.

---

## Item 11 — `Close()` on `AuthMiddleware` (gRPC Connection Leak)

### Problem

`services/api-gateway/internal/middleware/auth.go`:

```go
func NewAuthMiddleware(authAddr string) (*AuthMiddleware, error) {
    conn, err := grpc.NewClient(...)
    // conn is never stored; cannot be closed
    return &AuthMiddleware{client: authpb.NewAuthServiceClient(conn)}, nil
}
```

The `*grpc.ClientConn` is created and handed to the stub, but the reference is dropped. The gateway's shutdown sequence (`srv.Shutdown`) never drains or closes the gRPC connection, which means:
- In-flight auth RPC calls are not gracefully finished before the process exits
- The connection's goroutines (keepalive, transport) are leaked until the OS cleans up the process
- gRPC logs connection-reset errors on the auth-service side after gateway restarts

The same pattern exists in the per-service auth middlewares (`order-management-service/internal/middleware/auth.go`, etc.) but those are in separate processes with fewer restarts; the gateway is the priority because it hot-reloads.

### Decision

Store `conn *grpc.ClientConn` in `AuthMiddleware`. Add `Close() error` that delegates to `conn.Close()`. The gateway's `main.go` calls `defer authMW.Close()` in the shutdown sequence after `srv.Shutdown` returns (so in-flight requests finish before the gRPC channel closes).

No change to the middleware's behaviour for non-shutdown paths.

---

## Item 12 — Fix Documentation Filenames and Update `CLAUDE.md`

### Problem

Two documentation files have typos in their names:

| Actual filename | Intended filename |
|---|---|
| `docs/engneering-standards.md` | `docs/engineering-standards.md` |
| `docs/coding-guidelines..md` | `docs/coding-guidelines.md` |

`CLAUDE.md` references the intended names (correct spelling, single dot). Every Claude Code session therefore silently fails to load two of the four engineering standard documents — the AI instruction to "read and enforce" these files does nothing for two of them.

The double-dot filename (`coding-guidelines..md`) also causes issues on some filesystems and tools (e.g. glob patterns, some linters).

### Decision

**Rename both files** using `git mv` so history is preserved. Update `CLAUDE.md` to reference the new names (they already match — no change needed to CLAUDE.md content itself). Verify no other file references the old names.

---

## Item 13 — Integration Tests: Checkout Saga and Inventory Reservation

### Problem

Zero test coverage exists across all new services. The most critical paths — the checkout saga (order → reserve → payment → confirm) and the inventory atomic reserve (Lua → DB) — have no automated verification. These paths contain the most complex business logic and the most failure modes (compensation, rollback, idempotency replay).

### What to Test

**Checkout saga** — service-layer unit tests with mock contracts:
- Happy path: creates order, reserves stock, charges payment, confirms order
- Inventory failure: compensates already-reserved items, returns `INSUFFICIENT_STOCK`
- Payment failure: releases all reservations, marks order cancelled
- Payment not captured (non-CAPTURED status): same compensation as payment failure
- Idempotency: second call with same key returns cached order without re-running saga
- Validation: empty items, zero quantity, zero unit price, nil user/idempotency key

**Inventory reservation** — service-layer unit tests with mock repository + mock Redis:
- Redis hit (sufficient stock): Lua passes, DB write succeeds, returns reservation
- Redis miss: warms cache from DB, retries Lua, succeeds
- Redis says insufficient (result=0): returns `INSUFFICIENT_STOCK` without DB call
- Redis down: falls through to `dbReserve`, which hits DB directly
- DB rejects after Lua passes (race): Redis counter rolled back, error returned

### Approach

**Mock-based unit tests** at the service layer. The contracts interfaces (`OrderRepository`, `InventoryRepository`, `InventoryService`, `PaymentService`) are already defined — mock implementations can be written inline as test structs implementing those interfaces. No testcontainers, no running infrastructure required for the unit tests.

The tests live in `internal/service/` as `*_test.go` files in the same package (`package service`) for white-box access.

### Why not full integration tests hitting the DB?

Full integration tests (testcontainers + postgres + redis) would be the gold standard but require:
- Docker available in CI
- Test setup time (~5–10s per test run for container spin-up)
- Migration application in the test harness

That is the right long-term path (noted in LOW findings). For this priority we write service-layer unit tests which cover the saga logic. The `WithTransaction` wrapper and raw SQL queries are infrastructure — they're verified by the repository layer tests (future work, post-GA).

### Files created

```
services/order-management-service/internal/service/order_service_test.go
services/inventory-service/internal/service/inventory_service_test.go
```

---

## Implementation Order

1. **Item 12** — rename docs (zero risk, unblocks correct AI context in subsequent sessions)
2. **Item 11** — add `Close()` to `AuthMiddleware` (small, isolated)
3. **Item 10** — pagination for `ListOrders` / `ListSellerOrders` (touches service + repo + handler + contracts)
4. **Item 9** — create `pkg/relay/`, delete three service relay packages (largest refactor; touches `go.work` + three service `go.mod` + three `main.go`)
5. **Item 13** — write tests (can now import `pkg/relay` if needed; mocks reference the service contracts which are stable after item 10)

---

## Files Changed

| File | Item | Change type |
|---|---|---|
| `docs/engneering-standards.md` → `docs/engineering-standards.md` | 12 | Rename |
| `docs/coding-guidelines..md` → `docs/coding-guidelines.md` | 12 | Rename |
| `CLAUDE.md` | 12 | Reference update (may be no-op) |
| `services/api-gateway/internal/middleware/auth.go` | 11 | Add `conn` field + `Close()` |
| `services/api-gateway/main.go` | 11 | Call `defer authMW.Close()` |
| `services/order-management-service/internal/domain/contracts/repositories.go` | 10 | Add `OrderPageParams` |
| `services/order-management-service/internal/repository/order_repository.go` | 10 | Add LIMIT/OFFSET + COUNT |
| `services/order-management-service/internal/service/order_service.go` | 10 | Update signatures |
| `services/order-management-service/internal/handler/http/order_handler.go` | 10 | Parse params, return paginated |
| `pkg/relay/go.mod` | 9 | New file |
| `pkg/relay/relay.go` | 9 | New file (shared implementation) |
| `go.work` | 9 | Add `./pkg/relay` |
| `services/order-management-service/internal/relay/` | 9 | Deleted |
| `services/payment-service/internal/relay/` | 9 | Deleted |
| `services/inventory-service/internal/relay/` | 9 | Deleted |
| Three `go.mod` files | 9 | Add `pkg/relay` dependency |
| Three `main.go` files | 9 | Update import path |
| `services/order-management-service/internal/service/order_service_test.go` | 13 | New tests |
| `services/inventory-service/internal/service/inventory_service_test.go` | 13 | New tests |
