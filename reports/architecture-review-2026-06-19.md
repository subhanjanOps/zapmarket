# Architecture Assessment — ZapMarket

**Date:** 2026-06-19  
**Branch:** `features/cluster-setup`  
**Reviewer:** Principal Software Architect (Claude Code)

---

## Architecture Score: 6.2 / 10

The system has a credible microservices foundation — correct service boundaries, working outbox pattern, gRPC auth validation, Redis-backed service discovery, a circuit-breaking API gateway, and real business logic behind meaningful interfaces. However, the prescribed Clean Architecture layer structure is not implemented, DDD tactical patterns are largely absent, event contracts are unversioned, and observability is missing. The gap between the architecture documents and the actual code is the most urgent structural problem.

---

## Service Architecture

### Strengths

- **Hard service boundaries are respected.** No service reads another's database. Every cross-service call is gRPC (sync) or Kafka/outbox (async) — the principle is consistently applied.
- **Ownership is clear.** Each domain lives in one service: auth owns identity, product-catalog owns the catalog, order-management owns the checkout saga, inventory owns stock, payment owns the ledger.
- **The API Gateway is well-designed.** Dynamic route loading from PostgreSQL (with Postgres LISTEN/NOTIFY for instant propagation), Redis service discovery with static fallback, per-upstream circuit breakers, sliding-window rate limiting, audit logging, and CORS hardening are all present. The Swagger AutoBinder that self-registers services from their own Swagger docs is genuinely clever.

### Weaknesses

- **`order-management-service` is too large.** It owns the full checkout saga: validation, inventory reservation, payment charging, stock deduction, cancellation, and admin operations — all in one service. This is appropriate for an MVP, but the saga has synchronous dependencies on both inventory and payment, creating a long synchronous chain through three services.
- **Notification service has no domain.** It is a pure Kafka fan-out with a `switch` on raw event type strings. There is no domain model, no subscriber abstraction, no channel registry. When a fourth event type needs a different delivery channel (email vs push), there is nothing to extend cleanly.
- **gRPC clients on every per-service auth middleware are separate connections.** Each service (product-catalog, order-management) dials its own gRPC connection to auth-service. This is architecturally correct but means every service restart adds a connection. No connection pooling at the service mesh level.

---

## Clean Architecture Compliance

### Critical Finding: The Application Layer Does Not Exist

The prescribed layer structure (`domain/`, `application/`, `infrastructure/`, `interfaces/`) is described in both `docs/architecture-principles.md` and `docs/service-template.md`. The actual structure in every service is:

```
internal/
  domain/           ✅ exists (models + contracts/)
  service/          ⚠️  collapses application + domain services
  repository/       ⚠️  infrastructure without an infrastructure/ wrapper
  handler/http|grpc ⚠️  interface layer without interfaces/ wrapper
```

The `application/` layer (Use Cases, DTOs, Ports) is entirely absent. Business workflows live in `service/`, which is neither the prescribed `application/` nor a proper `domain/service/`. This means:

- No Use Case types: there is no `CheckoutUseCase`, `ReserveStockUseCase`, etc.
- No DTOs: handler code maps directly from HTTP request structs to domain models
- No Ports: `inventoryGateway` and `paymentGateway` interfaces live in the `service/` package, not in a `ports/` layer where they belong

**Verdict:** The architecture documents describe Clean Architecture; the code implements a three-layer architecture (domain + business + infrastructure). The two are incompatible and neither is fully documented.

### Dependency Rule — Partial Violations

1. **`auth-service`'s `AuthService` imports `goredis`** directly at the service layer (`auth-service/internal/service/auth_service.go:24`). A service-layer struct depending on an infrastructure SDK violates the dependency rule. `rdb` should be abstracted behind a `TokenBlacklist` port interface.

2. **`pkg/errors` imports `net/http` and `google.golang.org/grpc`** — a shared domain-level package pulling in transport frameworks. AppError should carry only its type/code/message; HTTP and gRPC mapping should live in transport-layer adapters.

3. **`SetRedis(rdb *goredis.Client)` post-construction injection** on `AuthService` violates the constructor DI principle stated in architecture-principles.md.

---

## DDD Assessment

### Entities

Present: `User`, `Order`, `OrderItem`, `Payment`, `Inventory`, `Reservation`, `Refund`, `Category`, `Product`, `SKU`. Correctly identified and live in `domain/models.go`.

**Problem:** They are anemic. Other than `Order.Transition()` (which is a proper domain behavior), the entities carry no behavior — they are data bags with JSON tags. All business rules live in `service/`.

### Value Objects

Essentially absent. Examples of where they should exist:

- `OrderStatus` is a `type string` constant — could be a Value Object with `IsTerminal()`, `CanTransitionTo()` methods
- `Money{amount int64, currency string}` — amount and currency appear together across Order, Payment, Refund, Ledger but are never co-typed
- `SellerStatus` in `User` is a `*string` — no type, no validation, no state machine
- `ReservationStatus`, `PaymentStatus`, `RefundStatus` are all `type string` with constants but no behavior

### Aggregates

Not formally modeled. `Order` + `[]OrderItem` is a clear aggregate but nothing enforces the invariant that items can only be created through the order. The repository operates on them independently with separate insert statements.

The `Inventory` row is an aggregate root (it owns `Reservation` rows), but `ReserveStock` and `GetReservationDetails` are separate repository methods with no aggregate root guarding consistency.

### Domain Services

`OrderService`, `InventoryService`, `PaymentService` are named as domain services but they are application-layer orchestrators: they call repositories, call external clients, manage caching, manage idempotency. They are use cases, not domain services.

A genuine domain service would be something like `StockReservationPolicy.CanReserve(inventory Inventory, qty int) bool` — pure domain logic with no infrastructure dependencies.

### Domain Events

**Not modeled.** Events are `map[string]string` payloads written to the `outbox` table as raw JSON. There is no typed `OrderConfirmedEvent`, `PaymentCapturedEvent`, etc. in the domain layer. The notification service matches on literal strings like `"order.confirmed"` with zero schema enforcement.

---

## Event Architecture

### Topics

Three topics: `orders`, `payments`, `inventory`. These are bucket topics — multiple event types flow through each. The architecture document prescribes versioned event names (`order.created.v1`), but:

- Topics are unversioned (`orders` not `orders.v1`)
- Event type is a header value, not part of the topic
- No schema registry (Avro/JSON Schema)
- The notification consumer `buildNotification()` matches 8 event type strings; if any producer changes a string, the consumer silently drops the notification

**This is the largest operational risk in the system.** A producer renaming `order.confirmed` to `order.confirmed.v2` will silently break the notification service with no error, no alert, no DLQ.

### Outbox Pattern

Correctly implemented. Status transitions in order, payment, and inventory write to an `outbox` table in the same transaction. `pkg/relay` polls and publishes. This is production-grade for Kafka-less or Debezium-less environments.

**Remaining risk in `pkg/relay`:** The relay marks a row `published_at = NOW()` *after* a successful publish. If the process crashes between the two operations, the event will be re-published on next startup. The consumer dedup (notification service uses Redis `SetNX`) handles this, but order-management and inventory have no consumer-side dedup. This is a latent exactly-once violation.

### AutoBinder Bug (Missed Fix)

The AutoBinder in `services/api-gateway/internal/registry/autobind.go:83` still calls `rdb.Keys()` — the blocking KEYS command. The CRIT-2 fix applied `SCAN` to `registry.go` but `autobind.go` was missed. Under high key counts this will block Redis's single-threaded command loop.

---

## Scalability Assessment

### Bottlenecks

1. **Auth-service is a synchronous chokepoint.** Every request to every service (via the gateway's `authMW.Authenticate`) makes a gRPC call to auth-service to validate the JWT. At 1,000 req/s across 5 services, auth-service receives 5,000 validation calls/second. No caching of valid tokens exists at the gateway layer. Token blacklisting requires Redis, but positive validation hits auth-service DB on every call (`GetUserByID`).

2. **The checkout saga is a synchronous long transaction.** A single checkout touches: Redis idempotency lock → DB insert → gRPC to inventory → DB update → gRPC to payment → DB update → Redis cache write. Five network hops synchronously before the HTTP response. P99 latency will be the sum of all five, and a slow payment service directly degrades the checkout endpoint.

3. **Per-request Redis SCAN in gateway route resolution.** `resolve()` is called on every proxied request. It calls `redisReg.Pick()`, which runs a full Redis SCAN. At 10,000 req/s this is 10,000 SCAN operations per second to Redis. The in-memory route table mitigates this somewhat, but the registry lookup is still live on every request.

### Failure Points

1. **Redis as single point of failure.** Rate limiting, service discovery, idempotency locking, token blacklisting, stock counters — all depend on Redis. There is no Redis Cluster or Sentinel configuration. A Redis failure cascades to: rate limiter down, auth blacklist down (revoked tokens accepted for their natural TTL), inventory Redis counter down (falls back to DB-only reserve which is less atomic).

2. **Kafka not running.** All three outbox relay instances attempt to publish to Kafka on startup. If Kafka is down, outbox rows accumulate. There is no alerting on `outbox WHERE published_at IS NULL AND created_at < NOW() - INTERVAL '5m'` — events can silently age without delivery.

3. **AutoBinder KEYS is still blocking** — Redis slowlog will fill during reconciliation cycles.

### Operational Risks

1. **Zero observability infrastructure.** `metrics.NewTracker()` in the gateway is an internal counter with no Prometheus endpoint. No service exposes `/metrics`. No OpenTelemetry traces. No correlation IDs propagated past the gateway. Diagnosing a latency spike requires reading structured logs across 6 services.

2. **No readiness probe.** Every service has `/health` returning 200 OK, but there is no `/ready` endpoint that checks DB connectivity and Redis reachability before accepting traffic. Kubernetes deployments will route traffic to a pod before its DB connection pool is warm.

3. **Migration on boot as a deployment coupling.** `MIGRATE_ON_BOOT=true` runs migrations synchronously in `main()`. With multiple replicas starting simultaneously, all will attempt `migrate.Up()` at the same time. Unless `pkg/migrate` uses a Postgres advisory lock, this is a race condition in multi-replica deployments.

---

## Technical Debt Inventory

| Priority | Item | Location | Risk |
|---|---|---|---|
| CRIT | `autobind.go` uses `rdb.Keys()` | `services/api-gateway/internal/registry/autobind.go:83` | Redis block under load |
| CRIT | Auth service hits DB on every token validation | `auth-service/internal/service/auth_service.go:169` | Auth-service DB is a hotspot |
| HIGH | Application layer is missing | All services | Architecture docs claim CA; code doesn't implement it |
| HIGH | No event schema / event versioning | `pkg/kafka/topics.go`, notification consumer | Silent event contract breaks |
| HIGH | No distributed tracing | All services | Latency diagnosis impossible |
| HIGH | No Prometheus metrics | All services | No SLO alerting possible |
| HIGH | Redis is a SPOF with no HA config | All services | Redis down = partial system failure |
| HIGH | Relay re-delivery has no consumer dedup in order/inventory | `pkg/relay/relay.go` | Potential duplicate order events processed |
| MED | No `application/` layer — Use Cases in `service/` | All services | Business workflows not independently testable |
| MED | Anemic domain models — no aggregate enforcement | `domain/models.go` | Invariants enforced only by convention |
| MED | No Value Objects for Money, Status | All services | Currency/amount mismatches not caught at compile time |
| MED | `pkg/errors` imports HTTP + gRPC | `pkg/errors/errors.go` | Transport leaks into shared domain package |
| MED | `AuthService.SetRedis()` post-constructor injection | `auth-service/internal/service/auth_service.go:210` | Violates DI principle |
| MED | `migrate.Up()` has no advisory lock | `pkg/migrate/migrate.go` | Multi-replica race on migration |
| LOW | No `/ready` readiness endpoint | All services | Kubernetes health checks too coarse |
| LOW | No DLQ / dead letter routing | `pkg/kafka` | Malformed events dropped silently |
| LOW | Notification service has no domain | `notification-service/` | Unextendable without rewrite |
| LOW | Integration test coverage near zero | All services | Only service-layer unit tests exist |
| LOW | No contract tests for gRPC or Kafka | All services | Schema breaks caught only at runtime |

---

## Refactoring Roadmap

### Phase 1 — Operational Safety (do first, before any traffic)

1. **Fix `autobind.go` KEYS → SCAN** (15 lines, zero risk) — `autobind.go:83`
2. **Add JWT validation caching at the gateway layer** — cache valid token hash → `{userID, role, exp}` in Redis for `min(TTL, 30s)`. Reduces auth-service load by ~95%
3. **Add `/ready` health endpoint** to all services checking DB ping + Redis ping
4. **Add advisory lock to `pkg/migrate`** using `SELECT pg_try_advisory_lock(1)` to prevent multi-replica migration races
5. **Fix `pkg/errors` transport coupling** — move `HandleHTTP` and `ToGRPCStatus` into `pkg/httpx` and `pkg/grpcx` respectively

### Phase 2 — Event Contract Hardening (before new consumers)

6. **Define typed event structs** in a `pkg/events` module: `OrderConfirmedEvent`, `PaymentCapturedEvent`, etc. — each with version field and JSON schema validation
7. **Rename topics to versioned names** or adopt event-type headers with version suffix: `order.confirmed.v1`
8. **Add consumer-side dedup** to order-management and inventory Kafka consumers
9. **Add outbox age alert** — health endpoint returning 503 when `published_at IS NULL AND created_at < NOW() - INTERVAL '5m'`

### Phase 3 — Observability (before production)

10. **Prometheus metrics** — instrument all services with `promhttp.Handler()` at `/metrics`; instrument: request count, latency histograms, DB pool stats, Redis op latency, circuit breaker state
11. **OpenTelemetry tracing** — propagate trace context from gateway through all gRPC and Kafka hops
12. **Redis HA** — move to Redis Cluster or add Sentinel; document fallback behavior for each Redis-dependent feature

### Phase 4 — Architecture Alignment (technical debt reduction)

13. **Extract `application/` layer** — introduce Use Case types (`CheckoutUseCase`, `ReserveStockUseCase`) that own the workflow, leaving `service/` as a thin facade or removing it
14. **Introduce Value Objects** — `Money`, typed status FSMs
15. **Fix `AuthService` Redis dependency** — introduce a `TokenBlacklist` port interface; `SetRedis()` disappears

---

## Recommended Next Steps

### Week 1 — Zero user-visible impact, highest risk reduction

1. Fix `autobind.go` KEYS → SCAN (15 lines, zero risk)
2. Add JWT caching at the gateway (reduces auth-service load 10×)
3. Add `/ready` endpoints (unblocks Kubernetes deployment)
4. Add migration advisory lock (prevents multi-replica corruption)
5. Add `pkg/events` typed event structs with a `v1` suffix convention

### Week 2 — Observability baseline

6. Prometheus metrics on all services
7. Outbox age health check as a Prometheus gauge + alert

### Month 2 — Architecture refactor

8. Introduce `application/` Use Case layer in order-management first, use it as the template for all others

---

## Final Verdict

**APPROVED FOR DEVELOPMENT — NOT PRODUCTION-READY.**

The foundations are sound: correct service boundaries, transactional outbox, idempotency, circuit breaking, and proper auth delegation. The team clearly understands distributed systems. What's missing is the operational layer — observability, Redis HA, event versioning, and the one remaining blocking Redis call — without which production incidents will be difficult to diagnose and some failures (Redis KEYS under load, silent event drops) will be difficult to detect at all. Two to three weeks of focused work on Phases 1 and 2 would make this system genuinely production-ready.
