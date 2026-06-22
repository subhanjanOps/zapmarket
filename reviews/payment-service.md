# payment-service Review

**Reviewer:** Principal Engineer
**Date:** 2026-06-22
**Branch:** features/cluster-setup

## Executive Summary

The payment-service is a well-structured, dual-transport service (gRPC for internal callers, HTTP for gateway webhooks only). It implements idempotent charge and refund flows, a double-entry ledger, transactional outbox publishing for all terminal payment states, HMAC-SHA256 webhook signature verification, and a `FakePaymentGateway` with a deliberate "magic amount" failure trigger. The domain model includes `Payment`, `LedgerEntry`, and `Refund` entities. The contracts package cleanly separates the `PaymentRepository` and `PaymentGateway` interfaces. There is no auth-service middleware (correct, as this service is internal-only). The main gaps are a missing test suite, an idempotency behavior inconsistency between payment-service and order-management-service under lock contention, and several smaller hardening items.

---

## Critical Findings

### C1 — Zero test coverage
**Scope:** Entire service
There are no test files in payment-service. The service implements non-trivial idempotency logic, a refund eligibility check, partial-refund detection, and two webhook idempotency guards — none of which are tested. Given that this service handles money movement and writes immutable ledger rows, an untested service is not acceptable for production.
**Fix:** At minimum, write unit tests for `PaymentService.ChargeCard` covering: happy path, validation errors, idempotent cache replay, idempotent DB replay, gateway decline, and partial/full refund. Mirror the pattern from `order-management-service`'s test file using miniredis.

---

## High Priority Findings

### H1 — `ChargeCard` returns a 409 Conflict under lock contention instead of waiting
**File:** `internal/service/payment_service.go` lines 86-91
When a second concurrent request arrives for the same idempotency key while the NX lock is held, payment-service immediately returns `pkgerrors.NewConflict("IDEMPOTENCY_CONFLICT", ...)`. By contrast, order-management-service spin-waits for the lock to be released and then re-reads the cache. The difference matters: order-management-service calls `ChargeCard` during its checkout saga, and if it retries after receiving a 409, it may charge the card twice if the retry arrives after the lock expires but before the DB record is written. The two services must use a consistent locking strategy — either both spin-wait or both return 409 and force the caller to implement retry-with-backoff.
**Fix:** Either replicate the spin-wait pattern from order-management-service, or document that `ChargeCard`'s caller must implement exponential backoff on IDEMPOTENCY_CONFLICT and add a test proving the behavior.

### H2 — `CreatePayment` is not inside a transaction but is followed by gateway I/O
**File:** `internal/repository/payment_repository.go` lines 32-47, `internal/service/payment_service.go` lines 121-127
`CreatePayment` issues a bare `INSERT` outside any transaction. If the process crashes after the INSERT but before `gateway.Charge` is called, there is a `PENDING` payment row with no gateway activity and no outbox event. This is technically acceptable (the row is orphaned in PENDING state), but more importantly: if `gateway.Charge` succeeds and then `MarkCaptured` fails, the money is captured at the gateway but the payment remains PENDING in the DB with no outbox event and no ledger rows. This is a silent funds-captured-but-not-recorded scenario.
**Fix:** Wrap `CreatePayment` + `gateway.Charge` + `MarkCaptured` in a saga with explicit compensation: if `MarkCaptured` fails after a successful charge, attempt a gateway refund and then `MarkFailed`. Log a CRITICAL alert if the refund also fails (manual reconciliation required).

### H3 — gRPC server has no auth middleware
**File:** `internal/handler/grpc/payment_grpc_handler.go`
The payment gRPC server accepts calls from any in-cluster client without verifying the caller's identity. While the service is not publicly routable, mTLS or at minimum a shared secret interceptor is required to prevent any rogue in-cluster service from initiating charges or refunds.
**Fix:** Add a gRPC `UnaryServerInterceptor` that verifies a shared service-to-service secret or enforces mTLS. This is consistent with the `grpcx.NewServer()` usage — add the interceptor there or in `main.go` when creating the server.

### H4 — `RefundPayment` has no idempotency protection
**File:** `internal/service/payment_service.go` lines 158-213
`ChargeCard` has Redis+DB idempotency guards. `RefundPayment` has none. A duplicate refund request (e.g., a retried HTTP call) will call `gateway.Refund` twice and create two `refunds` rows (assuming no DB-level unique constraint prevents it), resulting in double refunds.
**Fix:** Add an idempotency key to the `RefundPayment` signature (mirroring `ChargeCard`) and implement the same Redis-first / DB-fallback pattern. Add a `UNIQUE` constraint on `refunds(payment_id, amount)` as a last-resort DB guard (though a per-request idempotency key is the correct fix).

### H5 — `MarkCaptured` does not prevent double-capture at the DB level
**File:** `internal/repository/payment_repository.go` lines 49-83
The UPDATE statement does `WHERE id = $1 AND deleted_at IS NULL` with no status check. If `HandleCaptureWebhook` and a concurrent `ChargeCard` both call `MarkCaptured` for the same payment simultaneously, both can succeed: the second UPDATE will transition an already-CAPTURED payment to CAPTURED again and write two sets of ledger entries.
**Fix:** Add `AND status IN ('PENDING', 'AUTHORISED')` to the WHERE clause in `MarkCaptured` (consistent with the webhook handler's idempotency check in `HandleCaptureWebhook`). Return a no-op (not an error) if no rows are affected but the payment exists in a terminal state.

---

## Medium Priority Findings

### M1 — Outbox event missing for payment creation (PENDING state)
**File:** `internal/repository/payment_repository.go` `CreatePayment`
`MarkCaptured` and `MarkFailed` both write outbox events. `CreatePayment` does not. Downstream services cannot observe that a payment attempt was initiated — only that it resolved. A `payment.initiated` event would allow audit trails and timeout detection.
**Fix:** Add `insertOutboxRow(ctx, tx, id, "payment", "payment.initiated", payload)` inside `CreatePayment`, wrapped in a transaction.

### M2 — `LedgerEntry.Description` field is set on the struct but not mapped in the DB insert
**File:** `internal/repository/payment_repository.go` lines 197-201
`insertLedgerEntry` passes `e.Description` to the SQL, but the domain `LedgerEntry` struct (`internal/domain/models.go` line 55) does not include a `Description` field in its definition — it is set inline at the call site in `payment_service.go`. Cross-checking the INSERT column list with the struct is fragile. The struct should have a `Description string` field set before passing to the repository.
**Note:** On re-reading, `LedgerEntry` in payment-service domain does NOT have a Description field, but the repository `insertLedgerEntry` accepts a description parameter directly. The service passes the description inline. This means the description is never accessible after persistence via the domain model. Either add `Description string` to `LedgerEntry` or remove the field from the DB schema if it's not needed.

### M3 — Webhook handler does not return the correct HTTP status for already-processed events
**File:** `internal/handler/http/webhook_handler.go` lines 77-92
When `HandleCaptureWebhook` or `HandleFailureWebhook` is called for an already-resolved payment, the service layer logs and returns `nil` (idempotent no-op), and the HTTP handler returns `200 OK`. This is correct. However, there is no log entry at the HTTP handler level indicating an idempotent replay occurred. Gateway retry logic (Razorpay, Stripe) expects exactly `200 OK` for replays — the behavior is correct but should be explicitly documented.

### M4 — `FakePaymentGateway` magic failure amount (`amount % 100 == 13`) is undocumented at the API level
**File:** `internal/gateway/fake_gateway.go` lines 29-33
The magic decline trigger is only described in the file-level comment. It is not exposed via the OpenAPI spec or README. Developers testing the checkout flow from the frontend or API gateway will not know how to trigger a declined payment without reading this file.
**Fix:** Add a comment in the `.env.example` or a `docs/testing-guide.md` explaining the magic amount.

### M5 — No distributed tracing on gRPC handlers
Per the engineering standards, all services must support distributed tracing with `trace_id`. The gRPC server has no OpenTelemetry `UnaryServerInterceptor`. Incoming calls from order-management-service carry no trace context into payment-service.
**Fix:** Add `otelgrpc.UnaryServerInterceptor()` to `grpcx.NewServer()` or inject it in `main.go`.

---

## Low Priority Findings

### L1 — `PaymentStatus` and `RefundStatus` constants use mixed casing for `LedgerEntryType`
**File:** `internal/domain/models.go` lines 41-43
`LedgerDebit` and `LedgerCredit` are `"debit"` and `"credit"` (lowercase). All other status enums (`PaymentStatus`, `RefundStatus`) use UPPER_SNAKE_CASE per the project memory standard. The ledger entry type should follow the same convention.
**Fix:** Change to `LedgerDebit LedgerEntryType = "DEBIT"` and `LedgerCredit LedgerEntryType = "CREDIT"`. Update the DB migration accordingly.

### L2 — `GetTransaction` service method does not check if the caller is authorized to see the payment
**File:** `internal/service/payment_service.go` lines 216-221
`GetTransaction` takes a `paymentID` and returns the full payment record. No ownership check against `userID` or `orderID` is performed. Since this is an internal gRPC API, any authenticated in-cluster service can retrieve any payment. If this method is ever exposed to external consumers (e.g., via api-gateway), ownership must be enforced.
**Fix:** Add a `callerOrderID uuid.UUID` parameter and verify `payment.OrderID == callerOrderID` before returning.

### L3 — `webhook_handler.go` uses `http.Error` for error responses instead of the shared JSON envelope
**File:** `internal/handler/http/webhook_handler.go`
The webhook handler uses `http.Error(w, "message", statusCode)` which produces a plain-text response. The payment gRPC clients and any monitoring tooling expecting JSON will not parse these errors. Consistency with the rest of the codebase is preferred.

### L4 — `pkg/kafka.TopicPayments` is used but the constant source is in pkg — verify it matches the consumer's expected topic name
Minor: Verify that the Kafka topic constant used by payment-service and the topic consumed by notification-service (when implemented) match exactly.

---

## Recommended Refactoring Plan

**Sprint 1 (before any production traffic):**
1. Fix C1: Write unit tests for `ChargeCard` (all paths) and `RefundPayment`.
2. Fix H4: Add idempotency key to `RefundPayment` with Redis+DB guard.
3. Fix H5: Add `AND status IN ('PENDING', 'AUTHORISED')` to `MarkCaptured` WHERE clause.
4. Fix H2: Wrap `CreatePayment` + `MarkCaptured` in a compensating saga to handle gateway-captured-but-DB-failed scenarios.

**Sprint 2:**
5. Fix H1: Align lock-contention behavior between payment-service and order-management-service.
6. Fix H3: Add service-to-service auth interceptor on the gRPC server.
7. Fix L1: Standardize `LedgerEntryType` to UPPER_SNAKE_CASE.
8. Fix M1: Add `payment.initiated` outbox event in `CreatePayment`.

**Sprint 3:**
9. Add OpenTelemetry server interceptor (M5).
10. Add `GetTransaction` ownership check (L2).
11. Standardize webhook error responses to JSON (L3).

---

## Final Verdict

**Needs work before production.** The service architecture is sound: clean separation of gateway/repo/service/handler layers, correct dual-transport setup, proper HMAC webhook verification, and a functioning double-entry ledger. However, the absence of any tests in a money-handling service is a hard blocker. The double-capture vulnerability (H5), the unprotected refund path (H4), and the crash-between-charge-and-persist scenario (H2) are production-breaking correctness issues. These four items must be resolved before the service handles real transactions. The remaining items can follow in subsequent sprints.
