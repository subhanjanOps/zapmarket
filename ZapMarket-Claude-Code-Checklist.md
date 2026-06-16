# ZapMarket Architecture Audit & Refactoring Checklist

## Goal

Transform ZapMarket into a production-grade e-commerce microservices platform with:

- Clean Architecture
- Domain Driven Design principles
- gRPC internal communication
- Kafka event-driven workflows
- Outbox Pattern
- Shared infrastructure libraries
- Kubernetes deployment readiness
- CI/CD readiness

---

# Phase 1 — Monorepo Cleanup

## Shared Package Consolidation

- [x] Remove service-local config packages
- [x] Remove service-local crypto packages
- [x] Use root shared packages exclusively (auth-service and product-catalog-service now use pkg/database, pkg/grpcx, pkg/httpx, pkg/logger in addition to pkg/config/crypto/errors/proto)
- [x] Update imports across all services
- [x] Verify go.work dependency resolution

Target:

```text
pkg/
├── config
├── crypto
```

---

# Phase 2 — Shared Infrastructure Layer

Create:

```text
pkg/database
pkg/logger
pkg/httpx
pkg/grpcx
pkg/kafka
pkg/errors
```

### Database
- [ ] PostgreSQL connection helper
- [ ] Pool configuration
- [ ] Health checks
- [ ] Transaction helpers

### Logger
- [ ] Structured logging
- [ ] JSON output
- [ ] Correlation IDs
- [ ] Context-aware logging

### HTTP
- [ ] Response helpers
- [ ] Error mappers
- [ ] Middleware helpers
- [ ] Request ID middleware

### gRPC
- [ ] Server bootstrap
- [ ] Client factory
- [ ] Unary interceptors
- [ ] Recovery interceptor

### Kafka
- [ ] Producer wrapper
- [ ] Consumer wrapper
- [ ] Retry support
- [ ] Topic registration

---

# Phase 3 — Shared Protobuf Ownership

Target:

```text
pkg/proto
├── auth
├── catalog
├── inventory
├── order
├── payment
```

- [ ] Move all proto ownership to pkg/proto
- [ ] Remove duplicated proto definitions
- [ ] Centralize code generation
- [ ] Update imports

---

# Phase 4 — Domain Contract Cleanup

- [x] Move repository interfaces out of service package (both auth-service and product-catalog-service)
- [x] Define interfaces in domain/contracts layer (`internal/domain/contracts`)
- [x] Ensure services depend only on interfaces (auth-service's service layer previously depended on concrete `*repository.X` structs directly — now depends on `contracts.X` interfaces)

---

# Phase 5 — Error Handling Standardization

- [ ] Create shared AppError model
- [ ] Standardize error codes
- [ ] HTTP error mapping
- [ ] gRPC error mapping

Standard response:

```json
{
  "success": false,
  "code": "PRODUCT_NOT_FOUND",
  "message": "product not found"
}
```

---

# Phase 6 — Database Migrations

- [x] Add golang-migrate (new `pkg/migrate` module)
- [x] Create migration runner (`migrate.Up`/`migrate.Down` in `pkg/migrate`, run on boot via `MIGRATE_ON_BOOT`)
- [x] Add up/down migrations (auth-service, product-catalog-service — current schema captured as `0001_init`)
- [x] Integrate with Docker startup (Dockerfiles copy `migrations/` into the final image; `docker-entrypoint-initdb.d/init.sql` trimmed to just `CREATE DATABASE` for the two migrated services)

---

# Phase 7 — Product Catalog Hardening

### Category Filters
- [x] Typed filter structs (`domain.CategoryFilters`: parent ID, search, pagination, sort)
- [x] Pagination (shared `httpx.Paginated` envelope with `total`/`page`/`page_size`)
- [x] Sorting (`name|created_at|updated_at`, validated — invalid field returns 400)

### Product Filters
- [x] Category filtering
- [x] Seller filtering
- [x] Search support (name/description `ILIKE`)
- [x] Pagination
- [x] Sorting (`name|created_at|updated_at`, validated)

### SKU Filters
- [x] Product filtering
- [x] Active status filtering
- [x] Pagination
- [x] Sorting (`sku_code|price_amount|created_at|updated_at`, validated)

---

# Phase 8 — Inventory Service

- [x] Inventory CRUD (create/add side only — `AddStock`; no update/delete of inventory
      rows yet, not needed by any current consumer)
- [x] Stock reservation (`ReserveStock`, atomic, race-tested with 20 concurrent requests
      against a stock of 10 — no oversell)
- [x] Stock release (`ReleaseStock`, rejects double-release as a conflict)
- [x] Stock deduction (`DeductStock`, confirms a reservation as a permanent sale)
- [ ] Kafka integration (deferred to Stage 7 per `planning/03-inventory-service.md` —
      Order Management talks to Inventory via gRPC only until Kafka is turned on)

Events:

- inventory.reserved
- inventory.released
- inventory.deducted

---

# Phase 9 — Order Service

Workflow:

Order Created → Reserve Inventory → Process Payment → Confirm Order

- [ ] Order aggregate
- [ ] Order state machine
- [ ] Saga orchestration
- [ ] Compensation logic

States:

- PENDING
- RESERVED
- PAID
- CONFIRMED
- CANCELLED

---

# Phase 10 — Payment Service

- [ ] Payment creation
- [ ] Webhook handling
- [ ] Idempotency support
- [ ] Refund workflow

---

# Phase 11 — Notification Service

- [ ] Kafka consumers
- [ ] Email notifications
- [ ] SMS notifications
- [ ] Push notifications

---

# Phase 12 — Outbox Pattern

- [ ] Create outbox table
- [ ] Event publisher
- [ ] Debezium integration
- [ ] Remove direct Kafka publishing from transactions

Flow:

Business Data + Outbox Event → Debezium → Kafka

---

# Phase 13 — API Gateway

- [ ] Service discovery
- [ ] Routing
- [ ] JWT validation
- [ ] Rate limiting
- [ ] Request logging
- [ ] Correlation IDs
- [ ] Circuit breakers

---

# Phase 14 — Redis Layer

- [ ] Product cache
- [ ] Category cache
- [ ] Session cache
- [ ] Rate limit storage

Patterns:

- Cache Aside
- Write Through

---

# Phase 15 — Observability

- [ ] Prometheus metrics
- [ ] OpenTelemetry tracing
- [ ] Grafana dashboards
- [ ] Kafka metrics

---

# Phase 16 — Kubernetes Readiness

- [ ] Deployments
- [ ] Services
- [ ] ConfigMaps
- [ ] Secrets
- [ ] HPA
- [ ] Ingress

---

# Phase 17 — CI/CD

GitHub Actions:

### Pull Requests
- [ ] Lint
- [ ] Unit tests
- [ ] Security scans

### Main Branch
- [ ] Build images
- [ ] Push images
- [ ] Deploy

---

# Phase 18 — Production Readiness

### Security
- [ ] JWT rotation
- [ ] Refresh token revocation
- [ ] Secrets management
- [ ] TLS everywhere

### Reliability
- [ ] Retry policies
- [ ] Circuit breakers
- [ ] Dead letter queues
- [ ] Graceful shutdown

---

# Completion Criteria

- [ ] Shared infrastructure packages complete
- [ ] Centralized protobuf ownership
- [ ] Outbox pattern implemented
- [ ] Saga orchestration implemented
- [ ] OpenTelemetry integrated
- [ ] Kubernetes deployment ready
- [ ] CI/CD operational
- [ ] Redis caching enabled
- [ ] API Gateway operational
- [ ] Health/readiness endpoints implemented
- [ ] End-to-end order flow passes integration tests
