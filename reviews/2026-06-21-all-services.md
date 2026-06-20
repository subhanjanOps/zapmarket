# ZapMarket — Full Monorepo Code Review

**Reviewer:** Principal Engineer (7 parallel agents)
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Services Reviewed:** auth-service · product-catalog-service · order-management-service · inventory-service · payment-service · notification-service · api-gateway

---

## Executive Summary

ZapMarket demonstrates strong architectural ambition and several well-designed patterns: choreographed sagas, transactional outboxes, Redis Lua atomic operations, gRPC-based token validation, and circuit-breaking at the gateway. The engineering fundamentals are visible in the right places.

However, **every single service is CHANGES REQUIRED** and cannot be deployed to production in current state. The problems cluster into four systemic categories that cut across all services:

1. **Credentials committed to git** — every service has a `.env` file in source control
2. **No authentication on internal gRPC surfaces** — inventory, payment, and other services expose privileged write operations to unauthenticated callers
3. **Zero test coverage in 5 of 7 services** — only inventory-service has meaningful tests; order-management-service has partial coverage
4. **Production spin-loops** — both order-management-service and payment-service have `time.Sleep` busy-wait loops in hot paths that ignore context cancellation

---

## Verdicts by Service

| Service | Verdict | Critical | High | Medium | Low |
|---|---|---|---|---|---|
| auth-service | CHANGES REQUIRED | 5 | 7 | 8 | 8 |
| product-catalog-service | CHANGES REQUIRED | 4 | 7 | 8 | 6 |
| order-management-service | CHANGES REQUIRED | 4 | 6 | 8 | 6 |
| inventory-service | CHANGES REQUIRED | 4 | 5 | 7 | 6 |
| payment-service | CHANGES REQUIRED | 4 | 6 | 5 | 6 |
| notification-service | CHANGES REQUIRED | 4 | 6 | 7 | 5 |
| api-gateway | CHANGES REQUIRED | 4 | 6 | 10 | 9 |

---

## Cross-Cutting Critical Issues

These affect multiple services and should be addressed in a single coordinated effort.

### 1. Committed `.env` Files — All 7 Services
Every service has a `.env` file tracked in git containing `DB_PASSWORD=zappass123`, Redis URLs, JWT secrets, and webhook secrets.

**Immediate action:**
```bash
# For each service:
git rm --cached services/<name>/.env
echo "**/.env" >> .gitignore
git commit -m "security: remove committed .env files"
```
Then rotate any credential that may have reached a shared environment.

### 2. Committed Binary Artifacts
`inventory-service.exe`, `payment-service.exe`, `order-management-service.exe`, `notification-service.exe` are all tracked in git.

**Fix:** Add `**/*.exe` to root `.gitignore`.

### 3. No Authentication on Internal gRPC Surfaces
- **inventory-service** — `AddStock`, `ReserveStock`, `ReleaseStock`, `DeductStock` are open to anyone with network access
- **payment-service** — `ChargeCard`, `RefundPayment`, `GetTransaction` are open
- **api-gateway** — auth gRPC uses `insecure.NewCredentials()` (plaintext token transmission)
- **auth-service** — gRPC returns `codes.OK` for all errors, breaking downstream middleware

All internal gRPC services must add a unary interceptor that calls `auth-service.ValidateToken` (the pattern already exists in `product-catalog-service`).

### 4. Production Spin-Loops Ignoring Context
- `order-management-service/internal/service/order_service.go:118-131`
- `payment-service/internal/service/payment_service.go:87-101`

Both use `time.Sleep(50ms)` busy-polls up to 10 seconds, holding goroutines that ignore `ctx` cancellation. Replace with `ctx`-aware select or fail-fast 503.

### 5. Silent JSON Marshal Errors in Outbox Paths
- `inventory-service/internal/repository/inventory_repository.go:106, 154`
- `payment-service/internal/repository/payment_repository.go:76-81, 106-110`

```go
payload, _ := json.Marshal(...) // error silently discarded
```
Nil payload causes opaque DB errors or corrupt outbox events. Always check and return this error.

### 6. `go.mod` Declares `go 1.25.0` — Non-Existent Version
Multiple services. Go 1.25 does not exist. Will cause toolchain resolution failures on CI.

### 7. Concrete Infrastructure Types in Service Layer
Multiple services hold `*redis.Client` directly in service structs, violating Clean Architecture. Define cache interfaces in `domain/contracts/`.

### 8. Zero or Near-Zero Test Coverage
| Service | Test Files |
|---|---|
| auth-service | 0 |
| product-catalog-service | 0 |
| notification-service | 0 |
| payment-service | 0 |
| api-gateway | 0 |
| order-management-service | partial (Checkout only) |
| inventory-service | partial (ReserveStock only) |

---

## Top 10 Most Urgent Fixes (Across All Services)

Ranked by risk × blast radius:

1. **Remove all `.env` files from git and rotate credentials** — all 7 services
2. **Fix closure capture bug in api-gateway `buildRouter`** (`main.go:166`) — silently misroutes ALL traffic
3. **Add gRPC auth interceptor to inventory-service and payment-service** — privileged write operations are open
4. **Fix auth-service gRPC error codes** — downstream token validation middleware sees `codes.OK` on failures
5. **Replace spin-loops in order-management-service and payment-service** — goroutine/connection exhaustion under load
6. **Fix XFF spoofing in api-gateway** — rate limiting and IP blocklisting completely bypassable
7. **Fix seller authorization in product-catalog-service** — Seller A can modify/delete Seller B's products
8. **Fix `MarkCancelled` error discard in order-management-service** (`order_service.go:214`) — orders left in inconsistent state on payment failure
9. **Fix dedup correctness bug in notification-service** — duplicate notifications sent on Redis failure
10. **Fix `rows.Err()` missing in api-gateway audit handler** — truncated results returned as complete

---

## Service-Specific Highlights

### auth-service
- OAuth CSRF vulnerability (state not validated on callback)
- `OAuthService` instantiates new `AuthService` on every token issuance — Redis blacklist completely bypassed for OAuth users
- `AdminAuthMiddleware` skips the Redis blacklist check
- OAuth state generated with `time.Now()` instead of `crypto/rand`

### product-catalog-service
- Cache race condition in `DeleteProduct` — stale slug key leaks in Redis indefinitely
- Fetch-and-merge business logic inside HTTP handler instead of service layer
- gRPC `ClientConn` leaked — no way to close during graceful shutdown
- `GetImageByProductID` / `GetImageBySKUID` omit `object_key` from SELECT — latent data integrity bug

### order-management-service
- `MarkReserved` emits no outbox event — violates documented invariant; downstream consumers never see `order.reserved`
- Unbounded `?limit=` on all paginated queries — DoS vector
- Inventory released before `CANCELLED` persisted in DB — inconsistent state on DB failure
- `compensate` uses caller's context — client disconnect cancels cleanup calls

### inventory-service
- TOCTOU race in `ReleaseStock` — `GetReservationDetails` + `ReleaseStock` are two separate trips with no transaction
- Redis counter permanently drifts after `AddStock` + Redis failure
- No expiry sweep for abandoned `RESERVED` reservations — stock locked permanently
- `DeductStock` emits no outbox event

### payment-service
- Silent refund amount cap — caller requests X, receives Y with no error
- `CreateRefund` emits no outbox event — `payment.refunded` is invisible to downstream services
- `paymentSelectQuery` omits `gateway_response` and `deleted_at` from SELECT

### notification-service
- Monolithic `pkg/config` loaded — service carries irrelevant secrets (JWT, DB, OAuth, MinIO, payment webhook)
- Layer structure completely deviates from `architecture-principles.md`
- Retry backoff is fixed 5s for all 3 consumers — thundering herd on broker outage

### api-gateway
- Double-write to `ResponseWriter` when circuit opens after 5xx — corrupted HTTP responses
- `AutoBinder.seen` map unprotected — latent data race if `reconcile` called concurrently
- Router rebuilt every 1 second via polling despite LISTEN/NOTIFY providing instant notification
- `RedisRegistry.Pick` issues full Redis SCAN on every proxied request — thousands of Redis ops/sec at load

---

## Recommended Priority Order for Remediation

### Week 1 — Unblock Production
1. Purge `.env` and `.exe` files from git history across all services; rotate credentials
2. Fix api-gateway closure capture bug (30-minute fix, massive blast radius)
3. Fix api-gateway XFF spoofing
4. Add gRPC auth interceptors to inventory-service and payment-service
5. Fix auth-service gRPC error codes to return proper `status.Error`
6. Replace spin-loops in order-management-service and payment-service
7. Fix seller ownership check in product-catalog-service
8. Fix notification-service dedup correctness bug

### Week 2 — Data Integrity
9. Fix `MarkCancelled` error discard in order-management-service
10. Add `MarkReserved` outbox event in order-management-service
11. Add `payment.refunded` outbox event in payment-service
12. Add `inventory.deducted` outbox event in inventory-service
13. Fix `json.Marshal` error handling in inventory and payment outbox paths
14. Fix `rows.Err()` in api-gateway audit handler
15. Fix ReleaseStock TOCTOU — return `(skuID, qty)` from repo directly
16. Fix silent refund amount cap in payment-service

### Week 3 — Architecture & Quality
17. Fix `go 1.25.0` in all `go.mod` files
18. Define `IdempotencyCache` interface in payment-service and `cacheStore` in order-management-service
19. Fix product-catalog-service cache race in `DeleteProduct`
20. Fix notification-service to use service-specific config instead of `pkg/config`
21. Fix api-gateway shutdown order; add `rows.Err()` to all query loops
22. Begin writing unit tests — start with inventory-service (infrastructure in place) and order-management-service

### Ongoing — Test Coverage
Target coverage thresholds per service:
- Service layer: 80% line coverage
- Handler layer: 60% line coverage (httptest)
- Repository layer: integration tests via testcontainers

---

## Positive Patterns Worth Preserving

These design decisions are correct and should be carried forward to all services:

- **Transactional outbox** (inventory, order, payment) — write event + status change atomically
- **Redis Lua check-and-decrement** (inventory) — atomic stock reservation without distributed locks
- **gRPC-based token validation** (product-catalog-service) — correct auth delegation pattern
- **FSM for order state** (order-management-service `internal/domain/models.go`) — prevents illegal transitions
- **Circuit breaker per upstream** (api-gateway) — correct fault isolation
- **`sync/atomic.Pointer` for hot-reload** (api-gateway) — zero-downtime route updates
- **RBAC role middleware chaining** (product-catalog-service, order-management-service)
- **Constructor injection throughout** — no global state, no singletons

---

*Individual service reports: [`auth-service.md`](auth-service.md) · [`product-catalog-service.md`](product-catalog-service.md) · [`order-management-service.md`](order-management-service.md) · [`inventory-service.md`](inventory-service.md) · [`payment-service.md`](payment-service.md) · [`notification-service.md`](notification-service.md) · [`api-gateway.md`](api-gateway.md)*
