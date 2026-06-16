# ZapMarket — Upcoming Stages Overview

_Last reviewed: 2026-06-16_

This folder sequences the remaining work from `ZapMarket-Claude-Code-Checklist.md` into
buildable stages, ordered by dependency. Each stage file lists concrete tasks, file-level
targets, and a definition of done. Work top to bottom — later stages assume earlier ones
are merged (e.g. you cannot build Order Management's saga orchestration before Inventory
and Payment expose gRPC servers).

## Current state (verified against repo, 2026-06-16)

| Area | Status |
|---|---|
| `auth-service` | Implemented — HTTP (net/http ServeMux) + gRPC `ValidateToken`, JWT issuing |
| `product-catalog-service` | Implemented — chi router, categories/products/SKUs/images, gRPC handler |
| `order-management-service` | **Scaffold only** — `go.mod` + `main.go` placeholder |
| `inventory-service` | **Scaffold only** — `go.mod` + `main.go` placeholder |
| `payment-service` | **Scaffold only** — `go.mod` + `main.go` placeholder |
| `notification-service` | **Scaffold only** — `go.mod` + `main.go` placeholder |
| `pkg/config`, `pkg/crypto`, `pkg/database`, `pkg/errors`, `pkg/grpcx`, `pkg/httpx`, `pkg/logger` | Exist, used by auth + catalog |
| `pkg/proto` | Exists with `auth/` and `catalog/` proto packages centralized (most recent commits). `inventory/`, `order/`, `payment/` proto packages do not exist yet — created when those services are built |
| Kafka, Redis, Elasticsearch, MinIO, Debezium | Defined in `docker-compose.yml` but **commented out** — only Postgres runs today |
| Outbox pattern, saga orchestration, API gateway, observability, k8s, CI/CD | Not started |

## Stage sequence

1. **[Stage 1 — Foundation Hardening](01-foundation-hardening.md)** ✅ **Complete** — finished Phases 1, 4, 6 of the checklist (shared package adoption, domain contracts, migrations) before adding new services on top of shaky ground.
2. **[Stage 2 — Catalog Hardening](02-catalog-hardening.md)** ✅ **Complete** — Phase 7: filters, pagination, sorting on the one read-heavy service already live.
2b. **[Stage 2b — Product Image Upload via MinIO](02b-image-upload-minio.md)** ✅ **Complete** — not in the original checklist; bumped ahead of Stage 3 at explicit request. Real multipart upload for product images, backed by the MinIO bucket already provisioned in `docker-compose.yml`.
3. **[Stage 3 — Inventory Service](03-inventory-service.md)** ✅ **Complete** — Phase 8: build from scratch, gRPC `ReserveStock`/`ReleaseStock`, Postgres ledger.
4. **[Stage 4 — Payment Service](04-payment-service.md)** — Phase 10: build from scratch, gRPC `ChargeCard`, idempotent ledger.
5. **[Stage 5 — Order Management & Saga](05-order-management-saga.md)** — Phase 9 + 12: order FSM, saga orchestration calling Inventory/Payment, transactional outbox.
6. **[Stage 6 — Notification Service](06-notification-service.md)** — Phase 11: Kafka consumer-only service, the natural integration test of the event bus.
7. **[Stage 7 — Event Bus & Outbox Activation](07-event-bus-outbox.md)** — Phase 12 (cont.) + turn on Kafka/Debezium in docker-compose, wire `pkg/kafka`.
8. **[Stage 8 — API Gateway](08-api-gateway.md)** — Phase 13: single ingress, JWT validation, rate limiting, routing.
9. **[Stage 9 — Redis Caching Layer](09-redis-caching.md)** — Phase 14: turn on Redis, product cache, idempotency keys, inventory Lua scripts.
10. **[Stage 10 — Observability](10-observability.md)** — Phase 15: Prometheus, OpenTelemetry, Grafana.
11. **[Stage 11 — Kubernetes & CI/CD](11-k8s-cicd.md)** — Phases 16 + 17.
12. **[Stage 12 — Production Readiness](12-production-readiness.md)** — Phase 18: security hardening, reliability patterns, final completion criteria sign-off.

## Cross-cutting conventions established so far

- **Swagger/OpenAPI**: every service with a public HTTP API uses `pkg/swaggerx` +
  `swag init`-generated `docs/docs.go` (blank-imported), with the spec served at the
  literal path `/v1/docs/swagger.json` and the UI at `/v1/docs/*`. Never use
  `http.ServeFile` for the spec or `COPY docs/` in the Dockerfile — the spec is compiled
  into the binary. Full rationale in memory (`feedback_swagger_standard`). Apply this
  from day one in Stages 3-6, not as an afterthought.

## Why this order

- Inventory and Payment must exist with working gRPC servers **before** Order Management's
  saga can be built — Order is the orchestrator and has nothing to orchestrate otherwise.
- Notification is pure Kafka consumer, so it's the cheapest way to validate the event bus
  end-to-end once Order/Inventory/Payment are publishing real events.
- The outbox pattern is split into two passes: schema design lands in Stage 5 (where the
  first writer — Order — exists), and full Kafka/Debezium activation is deferred to Stage 7
  so we're not running infra services nothing yet talks to.
- API Gateway is sequenced after all services exist — there's nothing to route to before that.
- Redis, observability, k8s, and CI/CD are cross-cutting and deliberately pushed to the end so
  they're applied once across a stable surface instead of being half-retrofitted into every
  service as it gets built.

## How to use these files

Each stage file contains:
- **Goal** — what "done" means in one sentence.
- **Preconditions** — what must already be merged.
- **Tasks** — concrete, file-level checklist.
- **Out of scope** — explicitly deferred items, to prevent scope creep into later stages.
- **Definition of done** — verifiable exit criteria (tests, manual checks).

Update the status table above whenever a stage is completed, and check off the corresponding
section in `ZapMarket-Claude-Code-Checklist.md`.
