# Code Review: `notification-service`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

Deliberately lean Kafka consumer reading from 3 topics (orders, payments, inventory) and dispatching notifications. Readable scaffold with sound design ideas: `Notifier` interface, Redis dedup, at-least-once Kafka delivery, structured logging. However: no tests, a dedup correctness bug that causes duplicate notifications under Redis failure, a health server goroutine leak, and Kafka reader leaks on close errors block production deployment.

---

## Critical Findings

### CRIT-1: No Tests Exist At All
No `*_test.go` files anywhere. `buildNotification` is a pure function — trivially testable. `Handle` requires only interface mocks. Dedup branch and all event-type branches are completely unverified.

### CRIT-2: Duplicate Delivery Under Redis Failure
**File:** `internal/consumer/handler.go:34-41`
When `SetNX` returns an error, code falls through and processes the message anyway. On re-delivery with Redis still down, the notification is sent a second time — duplicate emails/SMS to real users.
**Fix:** Return `fmt.Errorf("dedup check: %w", err)` so the consumer doesn't commit the offset and retries after Redis recovers.

### CRIT-3: Kafka Consumer Reader Leaked on Close Error
**File:** `main.go:99-110`
`c.Close()` error is discarded with `_ =`. A failed close can leave connections open. Under sustained broker instability, a new `kafkago.Reader` is created on every retry cycle — connections accumulate until OS exhausts file descriptors.
**Fix:** Log errors from `c.Close()`.

### CRIT-4: Health Server Goroutine Has No Graceful Shutdown
**File:** `main.go:53-57`
`http.ListenAndServe` without an `http.Server` instance — no way to shut down when `ctx` is cancelled. Goroutine leaks on service stop.
**Fix:** Use `http.Server` with `Shutdown(ctx)` in the shutdown sequence.

---

## High Priority Findings

### HIGH-1: Monolithic Shared Config Is an Architectural Violation
**File:** `go.mod:9`, `pkg/config/config.go`
Notification-service loads `DBHost`, `DBPassword`, `JWTSecretKey`, `GoogleClientSecret`, `MinIOAccessKey`, etc. — none of which it uses. Should define its own `internal/config` with only `AppEnv`, `KafkaBrokers`, `RedisURL`, `HTTPPort`.

### HIGH-2: Layer Structure Doesn't Match `architecture-principles.md`
- `Notification` struct and `Notifier` interface belong in `internal/domain/`
- `LogNotifier` belongs in `internal/infrastructure/notifier/`
- `Handler` belongs in `internal/interfaces/kafka/`
- `buildNotification` is business logic embedded in the interface layer — must move to `internal/application/`

### HIGH-3: `buildNotification` Business Logic Lives in Interface Layer
**File:** `internal/consumer/handler.go:78-147`
Handler decides "what message does `payment.captured` produce?" — that is application logic. Coding guidelines: "Kafka Consumers must: 1. Deserialize. 2. Validate. 3. Call use case. Nothing else."

### HIGH-4: `outbox_id` Not Validated Before Use as Dedup Key
**File:** `internal/consumer/handler.go:29-33`
If `outbox_id` header is absent, dedup key becomes `"notif:dedup:"`. All header-less messages race on the same Redis key — only the first is processed, all others silently dropped.
**Fix:** Validate `outbox_id` and `event_type` are non-empty; return error if missing.

### HIGH-5: `google/uuid` and `pkg/registry` Are Unused Dependencies
**File:** `go.mod:6, 12`
Dead dependencies. Run `go mod tidy`.

---

## Medium Priority Findings

- **MED-1:** `payment.processed` and `payment.captured` produce identical notification text — likely copy-paste error; user receives duplicate emails for same payment
- **MED-2:** `notif.UserID` not validated before `Send` — empty UserID silently sends to wrong/no recipient
- **MED-3:** Health endpoint always returns `{"status":"ok"}` — doesn't probe Redis or Kafka for Kubernetes readiness
- **MED-4:** Retry backoff is fixed 5s for all 3 consumers — thundering herd on broker outage; need exponential backoff with jitter
- **MED-5:** `Notification` struct missing `Channel` and `Priority` fields — adding a real notifier later requires breaking changes
- **MED-6:** `LogNotifier.Send` ignores `ctx` without documentation — interface contract doesn't communicate cancellation expectation
- **MED-7:** `go.mod` declares `go 1.25.0` — Go 1.25 does not exist; will fail on standard CI

---

## Low Priority Findings

- **LOW-1:** Handler struct comment says "processes order events" — actually processes 3 topic types
- **LOW-2:** Port `:8085` hardcoded in `main.go:54` — should come from config
- **LOW-3:** `runConsumer` accepts `*consumer.Handler` — should accept `pkgkafka.HandlerFunc` for testability
- **LOW-4:** `defer rdb.Close()` at end of `main` — technically safe but fragile if goroutine tracking changes
- **LOW-5:** `Dockerfile` EXPOSE 8085 but `CLAUDE.md` service table lists notification-service port as `—`

---

## Recommended Refactoring Plan

**Phase 1 — Critical (Block Production):**
1. Fix CRIT-2: Return error from `Handle` when Redis `SetNX` fails
2. Fix CRIT-3: Log errors from `c.Close()` in `runConsumer`
3. Fix CRIT-4: Replace fire-and-forget `ListenAndServe` with graceful `http.Server`
4. Fix HIGH-4: Validate `outbox_id` and `event_type` at top of `Handle`
5. Write unit tests for `buildNotification` and `Handle`

**Phase 2 — Architecture:**
6. Create service-specific `internal/config` — remove dependency on monolithic `pkg/config`
7. Restructure packages: `Notification`/`Notifier` → `internal/domain/`, `LogNotifier` → `internal/infrastructure/`, `Handler` → `internal/interfaces/kafka/`, `buildNotification` → `internal/application/`
8. Run `go mod tidy` (removes `google/uuid`, `pkg/registry`)

**Phase 3 — Reliability:**
9. Implement exponential backoff with jitter in `runConsumer`
10. Add Redis ping to `/health` for readiness
11. Add `Channel` and `Priority` to `Notification` struct
12. Fix `go.mod` Go version
13. Resolve duplicate `payment.processed`/`payment.captured` notification text

---

## Final Verdict: CHANGES REQUIRED
