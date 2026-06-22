# notification-service Review

Reviewer: Principal Engineer (automated review via Claude Code)
Date: 2026-06-22
Branch: features/cluster-setup

---

## Executive Summary

The notification-service is a well-scoped Kafka consumer. Its core plumbing — graceful shutdown, per-topic consumer with backoff restart, Redis deduplication, and a clean notifier abstraction — is solid for an early-stage implementation. However, two correctness bugs (a zero-decimal currency division error and a dedup-before-send race that permanently drops notifications on transient errors), a significant SRP violation in handler.go, a test-escape hatch antipattern, and a missing infrastructure-level commit configuration prevent this from being production-ready.

**Verdict: CHANGES REQUIRED**

---

## Critical Findings

### CRIT-1 — Zero-decimal currency bug: divides by 100 when it must not

**File:** `internal/consumer/handler.go:36-38`

For zero-decimal currencies (JPY, KRW, IDR) the raw integer value stored by payment processors IS the whole unit — no sub-unit conversion is needed. The current code does `cents/100` for these currencies, silently collapsing every JPY amount by 100x. A payment of ¥1,500 (stored as `1500`) would be displayed as ¥15.

The test `TestFormatAmount_JPY_ZeroDecimal` passes `"150000"` and expects `"¥1,500"`, which is internally consistent with the /100 path but contradicts standard payment processor semantics (Stripe, Adyen) where JPY `1500` = ¥1,500.

**Required action:** Decide and document the minor-unit convention. If storing JPY as `1500` for ¥1,500 (standard), remove `/100` and update the test to pass `"1500"` expecting `"¥1,500"`.

---

### CRIT-2 — Dedup key set before Send: permanent notification loss on transient Send failure

**File:** `internal/consumer/handler.go:94-134`

The current execution order is:
1. SetNX dedup key in Redis (expires in 1 hour)
2. Parse payload
3. notifier.Send()
4. Return error on Send failure (offset not committed, message redelivered)

On retry after step 4:
1. SetNX returns false (key already exists)
2. Handler returns nil — treated as a dedup hit
3. Offset is committed
4. **Notification is permanently lost**

Any transient Send failure (network blip, downstream provider unavailable) silently drops the notification. This violates at-least-once delivery semantics.

**Required action:** Move SetNX to after a successful Send. Accept the theoretical risk of double-sending on a Redis write failure after Send succeeds — that is vastly preferable to silent permanent loss.

---

## High Priority Findings

### HIGH-1 — SRP violation: handler.go owns both event routing and monetary formatting

**File:** `internal/consumer/handler.go`

handler.go contains Kafka message routing (Handle, buildNotification), monetary string formatting (formatAmount, formatWithCommas), currency metadata tables (zeroDecimalCurrencies, currencySymbols), and an exported test-escape hatch (FormatAmountForTest).

Per the engineering standards (Single Responsibility Principle) and coding guidelines ("Kafka Consumers must: 1. Deserialize event, 2. Validate event, 3. Call use case — nothing else"), monetary formatting is a distinct concern that belongs in `internal/currency/format.go`.

**Required action:** Extract formatting into `internal/currency`. Delete `FormatAmountForTest`.

---

### HIGH-2 — FormatAmountForTest export is an antipattern

**File:** `internal/consumer/handler.go:68-70`

Wrapping a private function in a ForXxxTest export to allow external test packages to reach it pollutes the production API surface — any downstream code could call FormatAmountForTest from non-test files and the compiler will not prevent it.

Correct Go approaches: (a) move the function to its own package so tests import it naturally, or (b) change the test file to `package consumer` (whitebox) so it can call formatAmount directly.

**Required action:** Remove FormatAmountForTest. Use approach (a) or (b) above.

---

### HIGH-3 — Dedup TTL of 1 hour is too short

**File:** `internal/consumer/handler.go:95`

Kafka at-least-once delivery can redeliver messages after broker restarts or offset resets, which can happen hours or days later. A 1-hour TTL means any message redelivered after 60 minutes will bypass dedup and send a duplicate notification.

The rest of the codebase uses 24-hour idempotency key TTLs for order/payment endpoints.

**Required action:** Increase to at minimum `24 * time.Hour`, ideally `72 * time.Hour`. Add a comment documenting the rationale.

---

### HIGH-4 — Missing outbox_id format validation

**File:** `internal/consumer/handler.go:88-91`

The handler validates non-empty outbox_id but not its format. A whitespace-only or malformed value produces a Redis key like `notif:dedup:   ` that is valid at the Go/Redis level but semantically broken. outbox_id is expected to be a UUID per the transactional outbox pattern. `github.com/google/uuid` is already in go.mod.

**Required action:** Parse outbox_id with uuid.Parse() and return an error on failure.

---

### HIGH-5 — CommitInterval races with manual CommitMessages

**File:** `pkg/kafka/consumer.go:27`

`CommitInterval: time.Second` enables automatic background offset commits at 1-second intervals. The consumer simultaneously calls CommitMessages manually. These two commit paths can race: the background committer may auto-commit an offset before the handler returns successfully, destroying the at-least-once guarantee.

**Required action:** Set `CommitInterval: 0` in `pkg/kafka/consumer.go` to enforce manual-only commits.

---

## Medium Priority Findings

### MED-1 — No metrics or distributed tracing

Per docs/engineering-standards.md: "All services must support: Structured Logging, Metrics, Distributed Tracing" with required context fields trace_id, request_id, correlation_id.

The notification-service has only structured logging. No Prometheus counters (messages processed, notifications sent, dedup hits, errors by event type) and no trace ID propagation from Kafka headers.

**Required action:** At minimum, extract trace_id from Kafka message headers and attach to all log lines via slog.With. Add metrics as a follow-up backlog item.

---

### MED-2 — Health endpoint always returns 200 regardless of Redis/Kafka health

**File:** `main.go:48-57`

/health returns {"status":"ok"} unconditionally. If Redis is unavailable or Kafka consumers are in crash-restart loops, the health endpoint still reports healthy. Kubernetes liveness/readiness probes cannot detect the degradation.

**Required action:** Add Redis ping to a /readyz endpoint. Add a "last message processed" timestamp that the readiness check validates against a staleness threshold.

---

### MED-3 — buildNotification switch has no extensibility path

**File:** `internal/consumer/handler.go:140-204`

The function is 64 lines and within the 80-line limit but will grow as new event types are added. Each addition requires modifying this function, violating the Open/Closed Principle.

The pattern — each case produces a Notification struct from a string map — is a strong fit for a template table where new event types become data additions, not code changes.

**Required action:** Refactor to a template-driven lookup after HIGH-1 (currency extraction) is complete.

---

### MED-4 — inventory.reserved skip is indistinguishable from unknown event type in logs

**File:** `internal/consumer/handler.go:186-188`

The caller logs "no notification template for event" for both intentionally-skipped (inventory.reserved) and genuinely-unknown event types. Alerting on unhandled events is impossible.

**Required action:** Distinguish the two cases — either via a three-value return (send/skip/unknown) or by logging at a different level/key inside the inventory.reserved case.

---

### MED-5 — Per-topic consumer model should be documented

**File:** `main.go:84-90`

Three goroutines each create an independent Kafka reader with the same group ID but different topics. This is functionally correct but the design choice and its scaling implications (N instances = N readers per topic) are undocumented.

**Required action:** Add a comment in runConsumer documenting the per-topic-reader design decision.

---

## Low Priority Findings

### LOW-1 — go.mod declares go 1.25.0 which does not exist

**File:** `go.mod:3`

Go 1.25 does not exist. This is likely a typo for go 1.24.0. Can cause unexpected behaviour with `go work sync` and CI toolchain selection.

**Required action:** Correct to `go 1.24.0` or the actual minimum required version.

---

### LOW-2 — google/uuid in go.mod but unused in production code

**File:** `go.mod:6`

`github.com/google/uuid` is declared but not imported by any .go file. `go mod tidy` will remove it. Add it back when HIGH-4 (UUID validation) is implemented.

---

### LOW-3 — Test coverage covers only the happy-path of formatAmount

**File:** `internal/consumer/handler_test.go`

Current tests: 4 cases, all for formatAmount. Notably missing:
- Duplicate dedup key (skip path)
- Missing outbox_id or event_type header
- Malformed JSON payload
- buildNotification per event type
- inventory.depleted with empty seller_id
- notifier.Send failure and retry semantics
- Negative amounts in formatAmount

**Required action:** Add table-driven tests for Handle using miniredis and a mock notifier.Notifier.

---

### LOW-4 — pkg/registry in go.mod but not visibly used

**File:** `go.mod:13`

No import of `pkg/registry` visible in any .go file. Run `go mod tidy` to confirm if it is a transitive requirement or a leftover.

---

### LOW-5 — Health server port 8085 conflicts with planned import-service

**File:** `main.go:53`

Per project memory, the planned import-service targets port 8085. The notification-service health endpoint is already on :8085. Running both locally will produce a port conflict.

**Required action:** Move notification-service health to an unused port (e.g. 8087). Add HTTP_PORT to the config struct and .env.example.

---

## Recommended Refactoring Plan

Execute in this order to minimise conflict surface:

1. **Fix CRIT-2** — Move SetNX to after successful Send. Highest correctness impact.
2. **Fix HIGH-5** — Set CommitInterval: 0 in pkg/kafka/consumer.go. Restores at-least-once semantics.
3. **Fix CRIT-1** — Decide zero-decimal convention, fix formatAmount, update JPY test.
4. **Fix HIGH-3** — Increase dedup TTL to 72 * time.Hour.
5. **HIGH-1 + HIGH-2** — Extract currency formatting to internal/currency/format.go. Delete FormatAmountForTest.
6. **HIGH-4** — Add UUID validation for outbox_id.
7. **MED-3** — Refactor buildNotification to a template-table after extraction is complete.
8. **LOW-3** — Add handler integration tests with miniredis and mock notifier.
9. **LOW-1, LOW-2, LOW-4, LOW-5** — Go version typo, unused deps, port conflict.
10. **MED-1, MED-2** — Trace ID propagation and real readiness check.

---

## Final Verdict

**CHANGES REQUIRED**

Two correctness defects (CRIT-1 zero-decimal division bug, CRIT-2 dedup-before-send causes permanent notification loss on any transient Send failure) and one infrastructure issue (HIGH-5 auto-commit racing with manual commits) must be resolved before this service handles real payment events. The remaining findings are important quality and reliability improvements but do not independently block correctness.
