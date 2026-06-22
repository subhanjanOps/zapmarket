# order-management-service Review

**Reviewer:** Principal Engineer
**Date:** 2026-06-22
**Branch:** features/cluster-setup

## Executive Summary

The order-management-service is the most mature of the three reviewed services. It implements a full checkout saga with gRPC clients to inventory-service and payment-service, HTTP REST handlers (Chi router), Redis-backed idempotency with NX locking, transactional outbox publishing, an FSM-enforced order status model, RBAC auth middleware wired to auth-service, admin and buyer/seller route segregation, Swagger docs, graceful shutdown, and a solid service-layer test suite using miniredis. This is production-grade work overall, but two critical correctness bugs must be resolved before this service touches real money.

---

## Critical Findings

### C1 — Client-supplied unit prices are trusted without catalog validation
**File:** `internal/handler/http/order_handler.go` lines 33-34, 88, 91
`checkoutItemRequest.UnitPrice` is taken directly from the HTTP body and passed to the saga as-is. A buyer can submit `unit_price: 1` for any SKU and the service will create the order and charge ₹1. There is no validation against product-catalog-service. This is a financial integrity bug.
**Fix:** At checkout, fetch confirmed prices from product-catalog-service via gRPC before constructing `CheckoutItem.UnitPrice`. Never trust client-supplied monetary amounts.

### C2 — Broken `break` inside `select` in the idempotency spin-loop
**File:** `internal/service/order_service.go` lines 119-137
In the lock-wait loop, the `ticker.C` case executes a bare `break`. In Go, `break` inside a `select` exits only the `select` statement, not the surrounding `for` loop. The intended effect — exiting the loop when the lock is released — does not occur. The goroutine will spin for the full `lockTTL` (10 seconds) even after the lock has been released and the winner has populated the cache.
**Fix:** Use a labeled break (`break outer` on the `for` loop) or refactor to a named flag variable checked at the top of the loop.

---

## High Priority Findings

### H1 — gRPC connections to downstream services are never closed on shutdown
**Files:** `internal/clients/inventory_client.go`, `internal/clients/payment_client.go`, `internal/middleware/auth.go`
`grpc.NewClient` is called in each constructor, but the returned `*grpc.ClientConn` is consumed by the stub and never exposed. `main.go` does not close these connections during graceful shutdown. Under rolling restarts or repeated test runs this leaks file descriptors.
**Fix:** Return the `*grpc.ClientConn` from each constructor (or add a `Close() error` method to the client struct) and call `conn.Close()` in the shutdown sequence in `main.go`.

### H2 — All gRPC connections use insecure transport
**Files:** `internal/middleware/auth.go` line 8, `internal/clients/inventory_client.go` line 10, `internal/clients/payment_client.go` line 10
All three gRPC connections dial with `insecure.NewCredentials()`. Bearer tokens validated over auth-service and all inter-service calls travel over plaintext. In any environment beyond local dev this is a security gap.
**Fix:** Wire mTLS or at minimum TLS with server certificate verification. Accept a TLS config from environment variables; fall back to insecure only when `APP_ENV=development`.

### H3 — `parsePage` applies the wrong default when `limit=0` is passed explicitly
**File:** `internal/handler/http/order_handler.go` lines 226-242
The condition `if limit <= 0 || limit > 100 { limit = 100 }` treats `limit=0` (explicit) and `limit>100` (oversized) with the same outcome of 100. A caller that explicitly passes `limit=0` expecting the documented default of 20 receives 100 rows instead.
**Fix:** Separate default and cap: apply default 20 when no param is supplied (leave empty string check), then after parsing apply cap: `if limit > 100 { limit = 100 }`.

### H4 — `AdminListOrders` response shape is inconsistent with all other list endpoints
**File:** `internal/handler/http/admin_handler.go` lines 76-92
This handler defines an inline `listResp` struct and calls `JSON(w, ...)` directly, bypassing `httpx.Paginated`. The response envelope field names differ from those in `httpx.Paginated` (e.g., `page_size` vs. `limit`). API consumers receive different shapes from `/v1/orders` and `/v1/admin/orders`.
**Fix:** Replace the inline struct and `JSON` call with `httpx.Paginated(w, http.StatusOK, orders, total, page, limit)`.

### H5 — `Checkout` function is 187 lines, far exceeding the 80-line maximum
**File:** `internal/service/order_service.go` lines 81-268
The function conflates: idempotency cache check, NX locking, DB idempotency fallback, input validation, domain item construction, stock reservation loop, payment processing, stock deduction loop, outbox persistence, and cache population. Individual saga steps cannot be tested in isolation.
**Fix:** Extract named helpers: `checkIdempotencyCache`, `validateCheckoutItems`, `buildDomainItems`, `reserveAllItems`, `processPaymentOutcome`. Each should be 20-40 lines.

---

## Medium Priority Findings

### M1 — No request body size limit on the checkout endpoint
**File:** `internal/handler/http/base.go` `DecodeJSON`
`json.NewDecoder(r.Body)` has no `io.LimitReader`. A malicious client could send a body with thousands of items. The payment-service webhook handler correctly uses `io.LimitReader(r.Body, 1<<20)`.
**Fix:** Add `http.MaxBytesReader(w, r.Body, 64*1024)` in `DecodeJSON` or as a chi middleware.

### M2 — Order FSM `Transition` validates but never applies the state change
**File:** `internal/domain/models.go` lines 43-52
`Transition` returns nil on success but does not set `o.Status = next`. Every call site must manually assign status after calling `Transition`. This is an error-prone API; a future caller that forgets the assignment will silently leave the order in the wrong state.
**Fix:** After validation, set `o.Status = next` inside `Transition` before returning nil.

### M3 — `GetBySellerID` uses an unindexed `IN (SELECT DISTINCT ...)` subquery
**File:** `internal/repository/order_repository.go` lines 84-112
Without an index on `order_items(seller_id, order_id)`, this subquery will do a full scan of `order_items` at scale.
**Fix:** Add migration: `CREATE INDEX idx_order_items_seller_id ON order_items (seller_id, order_id)`.

### M4 — `MarkReserved` outbox payload uses `fmt.Sprintf` with `%q` instead of `json.Marshal`
**File:** `internal/repository/order_repository.go` line 182
```go
payload := []byte(fmt.Sprintf(`{"order_id":%q}`, orderID.String()))
```
`%q` produces Go string quoting semantics, not JSON. While UUIDs never contain characters that diverge, this is inconsistent with every other outbox payload in the codebase (which all use `json.Marshal`) and will silently produce malformed JSON for any future field that contains backslashes.
**Fix:** Replace with `json.Marshal(map[string]string{"order_id": orderID.String()})`.

### M5 — No `order.created` outbox event on initial order creation
**File:** `internal/repository/order_repository.go` `CreateOrder`
`MarkReserved`, `MarkConfirmed`, and `MarkCancelled` all write outbox events. `CreateOrder` does not. Downstream consumers (e.g., notification-service) cannot learn about new orders until they reach RESERVED state, which may be seconds later or never if reservation fails.
**Fix:** Add `insertOutboxEvent(ctx, tx, id, "order", "order.created", payload)` inside the `CreateOrder` transaction.

### M6 — No distributed tracing context injected into downstream gRPC calls
Engineering standards require `trace_id`, `request_id`, and `correlation_id`. The gRPC clients pass `ctx` from the HTTP handler but no OpenTelemetry `UnaryClientInterceptor` injects trace headers. Inter-service calls are invisible in distributed traces.
**Fix:** Add `otelgrpc.UnaryClientInterceptor()` to the dial options of `InventoryClient` and `PaymentClient`.

---

## Low Priority Findings

### L1 — `isAppError` helper in tests uses type assertion instead of `errors.As`
**File:** `internal/service/order_service_test.go` line 410
```go
ae, ok := err.(*pkgerrors.AppError)
```
If `AppError` is ever wrapped, this will fail to match. Use `errors.As(err, &ae)`.

### L2 — Swagger `@BasePath` is `/` but all routes live under `/v1`
**File:** `main.go` line 5
`// @BasePath /` causes Swagger UI try-it-out to construct URLs without the `/v1` prefix. Set `// @BasePath /v1` and strip the `/v1` prefix from all `@Router` annotations.

### L3 — Missing test coverage for `CancelOrder`, `GetSellerOrder`, and all admin operations
The test file covers `Checkout` comprehensively but leaves `CancelOrder`, `AdminCancelOrder`, `GetSellerOrder`, and `ListSellerOrders` with zero test cases.

### L4 — `context.WithoutCancel` requires Go 1.21+
**File:** `internal/service/order_service.go` line 408
Correct usage, but should be noted in any Go version compatibility documentation.

---

## Recommended Refactoring Plan

**Sprint 1 (before any production traffic):**
1. Fix C1: Add price validation against product-catalog-service at checkout.
2. Fix C2: Replace bare `break` with a labeled break in the idempotency spin-loop.
3. Fix H1: Expose and close gRPC `ClientConn` for all three connections in shutdown.
4. Fix H5: Extract `Checkout` into 4-5 named sub-functions.

**Sprint 2:**
5. Fix H2: Wire TLS for inter-service gRPC connections.
6. Fix H3/H4: Correct `parsePage` and align `AdminListOrders` response shape.
7. Fix M4/M5: Replace `%q` outbox payload; add `order.created` event.
8. Fix M3: Add `idx_order_items_seller_id` migration.

**Sprint 3:**
9. Add OpenTelemetry client interceptors (M6).
10. Add missing test cases for cancel/seller/admin paths (L3).
11. Fix Swagger BasePath (L2).

---

## Final Verdict

**Approved with required changes.** The service demonstrates strong engineering: a correct saga pattern, thoughtful idempotency at both Redis and DB layers, a clean FSM, proper auth wiring with RBAC, and genuine test coverage for the critical checkout path. Two critical bugs (client-supplied pricing and the broken spin-loop break) block production readiness. The high-priority items should be resolved in the same sprint. The remaining items are incremental hardening that can follow in subsequent sprints.
