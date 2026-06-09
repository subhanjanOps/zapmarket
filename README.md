# zapMarket Monorepo

This repository is structured as a Go monorepo using Go workspaces for service modules and shared packages.

## Layout

- `services/` — independent Go modules for each microservice
  - `auth-service`
  - `product-catalog-service`
  - `order-management-service`
  - `inventory-service`
  - `payment-service`
  - `notification-service`
- `pkg/` — shared libraries, proto definitions, tooling, and cross-service helpers
- `design.md` — the primary architecture reference for the system
- `AGENTS.md` — AI agent guidance and repository conventions

## Getting started

1. Ensure Go 1.22 or newer is installed.
2. Run:

   ```bash
   go work sync
   ```

3. Add service-specific implementation code inside `services/*`.
4. Add shared modules or proto artifacts inside `pkg/*`.

## Local dependencies

The project architecture depends on Kafka, Redis, PostgreSQL, Elasticsearch, MinIO, and Debezium CDC. Start these dependencies locally with:

```bash
docker compose up -d
```

The PostgreSQL init script also installs the `pgcrypto` extension in each service database for `gen_random_uuid()` support.

MinIO will be available at `http://localhost:9000` and its console at `http://localhost:9001`.

Then confirm the infrastructure is available before wiring the services into the stack.

## Module status

Each service is scaffolded as its own Go module, but currently contains only placeholder startup code.

## Next steps

- implement service APIs, gRPC clients/servers, and data access layers
- add shared proto definitions and generated Go code in `pkg/proto`
- update module `go.mod` files with required dependencies and replace placeholder commands
