# ZapMarket — Full Production Code Review
**Date:** 2026-06-21  
**Reviewer:** Claude (Principal Engineer, sequential review — no agents)  
**Scope:** All six backend services + three UI frontends  
**Standards applied:** `docs/engineering-standards.md`, `docs/architecture-principles.md`, `docs/coding-guidelines.md`

---

## Executive Summary

The codebase is architecturally sound. Clean Architecture is correctly observed: domain → repository → service → handler with no cross-layer leakage. The transactional outbox pattern, saga compensation, Redis Lua atomic reserve, circuit breaker, and sliding-window rate limiting are all implemented correctly and thoughtfully. The BFF cookie pattern across all three UIs is consistent and secure.

There are **one critical security defect**, **five high-priority bugs** (two of which cause silent data loss / broken UX), and several medium/low issues. No service has tests. The review verdict is **CHANGES REQUIRED**.

---

## Critical Findings

### CRIT-1 — auth-service `LoggingMiddleware` logs JWT tokens  
**File:** `services/auth-service/internal/handler/http/handlers.go`  
**Standard violated:** Engineering Standards §Security — "Never log tokens, passwords, or PII"

`LoggingMiddleware` wraps every response in a `strings.Builder` and logs the full body. On `/v1/auth/login` the body contains `{"access_token": "eyJ...", "refresh_token": "..."}`. Anyone with log access owns every session.

**Fix:** Strip the `responseRecorder` body capture entirely from logging, or log only status + byte-count. If request/response capture is needed for debugging, use a separate debug-only middleware gated by `APP_ENV=development`.

---

## High Priority Findings

### HIGH-1 — seller-ui BFF: cookie `maxAge` is 7 days; JWT expires in 1 hour  
**File:** `services/seller-ui/app/api/auth/login/route.ts:43`

```ts
maxAge: 60 * 60 * 24 * 7,  // 7 days
```

The auth-service issues access tokens with `JWT_ACCESS_EXPIRY_HOURS=1`. After one hour, all API calls return 401 but the `seller_token` cookie is still present and valid from the browser's perspective. Next.js middleware sees the cookie → does not redirect to login → the user sees a broken dashboard full of 401 errors with no explanation.

Compare with `backoffice-ui/app/api/auth/login/route.ts` which correctly uses `maxAge: 60 * 60` (1 hour).

**Fix:** Set `maxAge: 60 * 60` to match `JWT_ACCESS_EXPIRY_HOURS`. If long sessions are needed, implement a refresh-token flow in the BFF.

---

### HIGH-2 — payment-service: refund events never published to Kafka  
**File:** `services/payment-service/internal/repository/payment_repository.go:121-153`

`CreateRefund` writes to `refunds`, updates `payments.status`, and inserts ledger entries — but never calls `insertOutboxRow`. The `payment.refunded` event is never emitted.

Consequence: `notification-service` has a `payment.refunded` template (`handler.go:129`) that will never fire. Users never receive a refund confirmation.

**Fix:** Add `insertOutboxRow` at the end of the `CreateRefund` transaction with event type `payment.refunded`, including `payment_id`, `order_id`, `user_id`, `amount`, `currency`.

---

### HIGH-3 — inventory-service `AddStock`: Redis cache becomes inconsistently warm  
**File:** `services/inventory-service/internal/service/inventory_service.go:71`

```go
if redisErr := s.rdb.IncrBy(ctx, stockKey(skuID), int64(qty)).Err(); redisErr != nil {
```

`INCRBY` on a non-existent key creates it with value `qty` (the delta added). But the `luaReserve` script treats the key's value as `qty_available = qty_on_hand - qty_reserved`. If the key was absent, the written value is wrong: `qty` instead of `new_on_hand - qty_reserved`.

A subsequent `ReserveStock` call will read the incorrect value and may either over-reserve or reject valid requests.

**Fix:** After `AddStock` DB write succeeds, if the Redis key already exists (`SET ... XX`) increment it; if it doesn't exist, skip the increment — the next `ReserveStock` cache-miss will warm it correctly from DB.

```go
// Only increment if key already exists
s.rdb.SetArgs(ctx, stockKey(skuID), qty, goredis.SetArgs{XX: true, Get: true}) // or use a Lua check
```

---

### HIGH-4 — payment-service `ChargeCard`: idempotency lock wait is a busy-wait spin loop  
**File:** `services/payment-service/internal/service/payment_service.go:87-108`

```go
ticker := time.NewTicker(50 * time.Millisecond)
// polls every 50ms for up to 10 seconds = 200 Redis round-trips per blocked request
```

Under contention, each waiting goroutine hammers Redis with GET+EXISTS every 50ms for up to 10 seconds. With N concurrent duplicate requests, this is `N × 200` Redis ops. No exponential backoff.

**Fix:** Use exponential backoff with jitter (50ms → 100ms → 200ms → ..., capped at 1s), or simply return HTTP 409/429 immediately when the NX lock is held (the client retrying after a few seconds is safer than burning Redis).

---

### HIGH-5 — product-catalog-service `UpdateProduct`: read-modify-write race  
**File:** `services/product-catalog-service/internal/handler/http/product_handler.go:270-316`

```go
existingProduct, _ := h.productService.GetProductByID(...)  // READ
// merge fields
h.productService.UpdateProduct(...)                          // WRITE
```

Two concurrent partial updates (e.g., seller changes name, admin changes status simultaneously) can interleave: both read the same `existingProduct`, the second write silently overwrites the first. No optimistic lock, no row version, no conditional UPDATE.

**Fix:** Add `updated_at` / `version` to the UPDATE's WHERE clause (`WHERE id = $1 AND updated_at = $2`), return 409 if rows affected = 0.

---

## Medium Priority Findings

### MED-1 — payment-service `MarkCaptured`/`MarkFailed`: extra SELECT inside transaction  
**File:** `services/payment-service/internal/repository/payment_repository.go:74, 107`

After `UPDATE payments SET status = 'CAPTURED'`, a second `SELECT order_id, user_id FROM payments WHERE id = $1` is issued within the same transaction to populate the outbox payload. The data was available before the transaction; passing it in as a parameter avoids a round-trip.

**Fix:** Extend `MarkCaptured(ctx, paymentID, gatewayTxnID, orderID, userID, entries)` signature, or use `RETURNING order_id, user_id` on the UPDATE.

---

### MED-2 — notification-service: duplicate handlers for `payment.processed` and `payment.captured`  
**File:** `services/notification-service/internal/consumer/handler.go:105-119`

Both `payment.processed` and `payment.captured` emit identical "Payment successful" notifications. `MarkCaptured` in the payment repository publishes `payment.processed` (line 86). If `HandleCaptureWebhook` is ever wired in, it will call `MarkCaptured` again, publishing a second `payment.processed`. Users would receive duplicate success notifications.

**Fix:** Pick one canonical event name (`payment.captured`). Remove the `payment.processed` template or alias them to the same handler.

---

### MED-3 — api-gateway `getStats` silently swallows row errors  
**File:** `services/api-gateway/internal/admin/handler.go`

```go
_ = h.db.QueryRowContext(...).Scan(&total)
```

All count queries in `getStats` discard scan errors with `_`. A schema mismatch or connection error would return zero counts with HTTP 200, with no log entry.

**Fix:** Log the error at WARN level and return the count as `-1` or include an `errors` field in the stats response.

---

### MED-4 — auth-service `AuthResponse` conflates success and error  
**File:** `services/auth-service/internal/handler/http/handlers.go`

A single `AuthResponse` struct carries `User`, `AccessToken`, `RefreshToken`, and `Error`. Callers must inspect the `Error` field even on HTTP 200. This is an untyped discriminated union pattern — error-prone to consume and inconsistent with other services' `{"code":..., "message":...}` error shape.

**Fix:** Return a typed success response `{user, access_token, refresh_token}` on 200; use `ErrorResponse()` on non-200 paths.

---

### MED-5 — `inventory.depleted` notification relies on `seller_id` in payload  
**File:** `services/notification-service/internal/consumer/handler.go:142-149`

```go
case "inventory.depleted":
    UserID: payload["seller_id"],
```

The `inventory-service` outbox currently only emits `inventory.reserved` and `inventory.released` events. There is no `inventory.depleted` producer anywhere in the codebase. If this event is wired up later without including `seller_id` in the payload, the notification will fire with an empty `UserID` — silently dropped or delivered to no one.

**Fix:** Document the required payload schema for `inventory.depleted` as a contract, or add a payload validation guard (`if payload["seller_id"] == "" { return nil }`).

---

## Low Priority Findings

### LOW-1 — auth-service: dead method-check guards  
**File:** `services/auth-service/internal/handler/http/handlers.go`

```go
if r.Method != http.MethodPost { ... }
```

Go 1.22+ `mux.HandleFunc("POST /path", h)` already enforces the method. The explicit check is dead code and a maintenance hazard.

---

### LOW-2 — seller-ui: no auth hint cookie  
**File:** `services/seller-ui/app/api/auth/login/route.ts`

`backoffice-ui` sets a non-httpOnly `bo_auth_hint` cookie so client JS can detect login state without reading the `bo_token`. `seller-ui` omits this. Client components have no way to detect session state without an API round-trip.

---

### LOW-3 — api-gateway: misplaced comment  
**File:** `services/api-gateway/internal/proxy/proxy.go`

The comment describing `originAllowed()` appears after the `statusRecorder` struct instead of above the function. Minor readability issue.

---

### LOW-4 — Zero test coverage across all services  
No service has a single test file. The saga, Redis Lua path, idempotency, and outbox are all untested. Engineering standards require unit coverage of service-layer logic and integration coverage of repository methods.

---

### LOW-5 — seller-ui login: `cache: "no-store"` retained  
**File:** `services/seller-ui/app/api/auth/login/route.ts:18`

`backoffice-ui` had `cache: "no-store"` removed in a prior fix (Next.js 16 patched fetch). `seller-ui` still has it. Low risk since it's on a POST, but inconsistent.

---

## Recommended Refactoring Plan

**Sprint 1 (Security / Data integrity):**
1. CRIT-1: Remove response body logging from `LoggingMiddleware`
2. HIGH-2: Add `insertOutboxRow` to `CreateRefund`
3. HIGH-1: Fix seller-ui cookie `maxAge` to 1 hour

**Sprint 2 (Correctness):**
4. HIGH-3: Fix `AddStock` Redis increment to use `XX` flag (only update existing key)
5. HIGH-5: Add optimistic locking to `UpdateProduct`
6. HIGH-4: Replace spin-wait with backoff or immediate 409

**Sprint 3 (Quality):**
7. MED-1: Eliminate extra SELECT in `MarkCaptured`/`MarkFailed`
8. MED-2: Deduplicate payment notification templates
9. MED-3: Log (don't swallow) stat query errors
10. MED-4: Split `AuthResponse` into typed success + error shapes

**Ongoing:**
- Add service-layer unit tests (mock repos via interfaces already defined)
- Add repository integration tests against a test PostgreSQL schema

---

## Final Verdict

**CHANGES REQUIRED**

The architecture is solid and the complex distributed system primitives (outbox, saga, Lua reserve, circuit breaker) are implemented correctly. However, CRIT-1 is a live token leak in any environment where logs are readable by more than one person, HIGH-2 is silent data loss (refund events never fire), and HIGH-1 creates a broken UX within one hour of a seller logging in. These must be fixed before any production deployment.
