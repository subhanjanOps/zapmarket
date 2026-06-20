# Code Review: `payment-service`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

Architecturally sound with correct transactional patterns: double-entry ledger, transactional outbox, HMAC-SHA256 webhook signature verification, Redis idempotency caching, and graceful shutdown. However: committed `.env`, completely unauthenticated gRPC surface, a production spin-loop, silent JSON marshal errors in the outbox path, a silent refund amount cap, and zero tests block production deployment.

---

## Critical Findings

### C-1: `.env` File Committed With Credentials
**File:** `services/payment-service/.env`
Contains `DB_PASSWORD=zappass123` and `PAYMENT_WEBHOOK_SECRET`. Must be removed from git history and `.gitignore`d.

### C-2: No Authorization on Any gRPC Method
**Files:** `internal/handler/grpc/payment_grpc_handler.go:23-87`, `main.go:91-93`
`ChargeCard`, `RefundPayment`, `GetTransaction` are completely open — any caller with network access can charge a card or issue a refund against any `user_id`/`order_id`. No JWT validation interceptor.

### C-3: Busy-Wait Spin Loop on Hot Idempotency Path
**File:** `internal/service/payment_service.go:87-101`
10-second `time.Sleep(50ms)` poll loop. Ignores `ctx` cancellation — goroutines stay parked up to 10 seconds even after client disconnect. Goroutine and Redis connection exhaustion under load.

### C-4: Silent JSON Marshal Errors Produce Corrupt Outbox Payloads
**File:** `internal/repository/payment_repository.go:76-81, 106-110`
```go
payload, _ := json.Marshal(...)
```
If `payload` is nil, `INSERT INTO outbox ... JSONB NOT NULL` fails with an opaque DB error instead of a meaningful application error.

---

## High Priority Findings

### H-1: Silent Amount Cap on Refund — Data Integrity Hazard
**File:** `internal/service/payment_service.go:183-185`
```go
if amount > payment.Amount { amount = payment.Amount }
```
Caller requests refund of X, receives refund of Y with no error. Silent data discrepancy. Should return a validation error instead.

### H-2: No `ReadHeaderTimeout` — Slowloris Vulnerability
**File:** `main.go:82-88`
`ReadTimeout` and `WriteTimeout` set, but `ReadHeaderTimeout` absent. Attacker sending partial headers holds connection until `ReadTimeout`.

### H-3: `MarkCaptured` Performs Redundant Read Inside Transaction
**File:** `internal/repository/payment_repository.go:72-75`
SELECT after UPDATE to get `order_id`/`user_id` for outbox payload. Use `RETURNING order_id, user_id` on the UPDATE instead.

### H-4: `ChargeCard` Hardcodes `Gateway: "fake"` in Service Layer
**File:** `internal/service/payment_service.go:129`
Gateway name is a domain concern. Add `Name() string` to `PaymentGateway` interface; remove string literal from service.

### H-5: No Tests Whatsoever
No `*_test.go` files exist anywhere. All idempotency logic, ledger entries, webhook verification, and refund state machine are untested. Infrastructure for mocking is in place but unused.

### H-6: `rdb *goredis.Client` Injected as Concrete Type Into Service Layer
**File:** `internal/service/payment_service.go:38-45`
Service layer depends on infrastructure implementation. Define `IdempotencyCache` interface in `internal/domain/contracts/`; implement with Redis adapter.

---

## Medium Priority Findings

- **M-1:** `LedgerEntryType` uses `"debit"`/`"credit"` (lowercase) — inconsistent with project standard of `UPPER_SNAKE_CASE` for all enum strings
- **M-2:** `ChargeCard` is 114 lines — exceeds 80-line maximum; extract `resolveIdempotency` private method
- **M-3:** Health endpoint returns `{"status":"ok"}` without probing DB or Redis
- **M-4:** `CreateRefund` writes no outbox event — `payment.refunded` is a core business fact; downstream services have no way to react to refunds
- **M-5:** `paymentSelectQuery` omits `gateway_response` and `deleted_at` — fields always zero/nil in code despite existing in DB schema

---

## Low Priority Findings

- **L-1:** `payment-service.exe` binary committed to repo
- **L-2:** `go.mod` declares `go 1.25.0` — Go 1.25 does not exist
- **L-3:** `payment_taxes` table exists in schema but has no domain model, repository, or service logic
- **L-4:** `pkg/registry` listed in `go.mod` but never imported
- **L-5:** `PaymentService` interface defined in `service` package alongside implementation — should be in `domain/contracts/`
- **L-6:** `FakePaymentGateway` ignores `ctx` without `_ = ctx` annotation — linter will flag

---

## Recommended Refactoring Plan

**Phase 1 — Immediate (Security / Data Integrity):**
1. Remove `.env`, `.exe` from git; rotate credentials
2. Add gRPC auth interceptor — call auth-service `ValidateToken` RPC
3. Replace silent refund cap with validation error
4. Fix `json.Marshal` error handling in `MarkCaptured` and `MarkFailed`

**Phase 2 — Short-term (Reliability):**
5. Replace spin-loop in `ChargeCard` with `ctx`-aware ticker select
6. Add `ReadHeaderTimeout: 5s` to HTTP server config
7. Extract idempotency block from `ChargeCard` into `resolveIdempotency`
8. Define `IdempotencyCache` interface; remove concrete `*goredis.Client` from service
9. Add `payment.refunded` outbox event in `CreateRefund`
10. Fix `paymentSelectQuery` to include `gateway_response` and `deleted_at`

**Phase 3 — Medium-term (Quality):**
11. Write unit tests for all idempotency paths, refund state machine, webhook signature verification
12. Update `LedgerEntryType` to `UPPER_SNAKE_CASE` with DB migration
13. Add `Name()` to `PaymentGateway` interface; remove `"fake"` literal from service
14. Move `PaymentService` interface to `internal/domain/contracts/`
15. Fix health endpoint to probe DB and Redis
16. Run `go mod tidy`; fix `go 1.25.0` version

---

## Final Verdict: CHANGES REQUIRED
