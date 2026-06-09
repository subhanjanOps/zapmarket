# Agent Guide for zapMarket

## Purpose
This repository currently contains a high-level ecommerce microservices architecture design document (`design.md`). The main goal for an AI coding agent is to understand the architecture and avoid assuming existing implementation details that are not present in the repo.

## Key Project Concepts
- Six backend services: User/Auth, Product Catalog, Order Management, Inventory, Payment, Notification
- Communication:
  - gRPC for synchronous internal service-to-service calls
  - Kafka for asynchronous events and service integration
- Redis used for caching, distributed locks, idempotency, rate limiting, and deduplication
- Each service owns its own data store; no cross-service joins
  - PostgreSQL for most services, including Product Catalog
  - Elasticsearch for full-text and faceted search
- Order Management is the saga orchestrator and uses an outbox pattern for event publishing

## Important Files
- `design.md` — the authoritative architecture and integration reference for this repo
- `go.work` — root Go workspace for service modules
- `services/*` — per-service Go modules for each microservice
- `pkg/proto` — shared proto/package placeholder for cross-service code

## How to Use This Guide
- Use `design.md` as the primary source of truth for architecture, topic names, and communication patterns
- Do not infer untracked source code, build scripts, or service implementation details beyond what is described in `design.md`
- If asked to implement or modify code, confirm whether actual service source files or repo structure exist before generating concrete changes
- If new code is added later, update this guide to reference the relevant service directories and any build/test conventions

## When suggesting changes or generating code
- Prefer architectural alignment with the existing microservices patterns:
  - gRPC for sync calls between services
  - Kafka topics for async events
  - Redis for caches, locks, idempotency, and deduplication
- Keep service ownership clear: each service owns its own database and event topics
- Preserve the intended flow of checkout / order saga and compensating transactions described in `design.md`

## Notes
- No build or test commands are defined in this repo yet
- No code or service directories are currently present; treat this repo as an architecture/design repository until additional source files appear
