# Stage 1 — Foundation Hardening

**Status: ✅ Complete (2026-06-16)**

Corresponds to checklist **Phase 1 (remainder), Phase 4, Phase 6**.

## Goal

Close the gaps in shared-package adoption and domain-contract layering in the two live
services (`auth-service`, `product-catalog-service`) and add a real migration system,
so the four new services built in later stages inherit a consistent, finished pattern
instead of copying half-migrated conventions.

## Preconditions

None — this is the next unit of work on top of the current `main`.

## Tasks

### 1.1 Shared package adoption audit (Phase 1 remainder)
- [x] Grep both live services for any service-local `config` or `crypto` package that
      duplicates `pkg/config` / `pkg/crypto` and delete the duplicate, repointing imports.
      (None found duplicating config/crypto; catalog's `internal/errors` and
      `internal/handler/http/base.go` duplicated `pkg/errors`/`pkg/httpx` logic instead —
      `base.go` now delegates to `pkg/httpx`.)
- [x] Confirm `auth-service` and `product-catalog-service` both import `pkg/database`,
      `pkg/errors`, `pkg/grpcx`, `pkg/httpx`, `pkg/logger` rather than ad-hoc equivalents.
- [x] Run `go work sync` and `go build ./...` from repo root after the sweep.

### 1.2 Domain contract cleanup (Phase 4)
- [x] `services/product-catalog-service/internal/service/repository_interfaces.go` moved to
      `internal/domain/contracts/repositories.go`.
- [x] Confirm `internal/service/*.go` depends only on the interface types, not on concrete
      `internal/repository` structs.
- [x] Repeat the same audit for `auth-service` — found it had **no** interfaces at all
      (service layer depended directly on `*repository.UserRepository` etc.); added
      `internal/domain/contracts/repositories.go` and rewired `AuthService`/`OAuthService`.
- [x] Template established in `internal/domain/contracts` for Stages 3-6 to follow.

### 1.3 Database migrations (Phase 6)
- [x] Added `pkg/migrate`, a thin wrapper module around `golang-migrate/migrate/v4`
      (DSN-based, no DB-pool internals needed).
- [x] Created `migrations/0001_init.up.sql` / `.down.sql` for both `auth-service` and
      `product-catalog-service`, capturing the current schema. Trimmed
      `docker-entrypoint-initdb.d/init.sql` down to just `CREATE DATABASE` statements for
      these two services (inventory/order/payment schemas left in init.sql until those
      services are built in Stages 3-5).
- [x] Decided run-on-boot over a separate `cmd/migrate`: `migrate.Up` runs from each
      service's `main.go`, guarded by `MIGRATE_ON_BOOT` (default `true`).
- [x] Wired into Docker: both Dockerfiles now copy `migrations/` into the final image so
      `MIGRATE_ON_BOOT` works in containers too, not just local `go run`.
- [x] Verified live: started Postgres + both services, confirmed `schema_migrations` was
      populated and `/health` and `/api/v1/categories` responded correctly.

## Out of scope
- New service scaffolding (Stages 3-6).
- Outbox table — that's introduced in Stage 5 alongside Order Management, using the
  migration system built here.

## Definition of done
- `go build ./...` and `go vet ./...` pass clean from repo root.
- No `internal/service` package in either live service defines a repository interface.
- Both services can be torn down (`docker compose down -v`) and brought back up with
  schema fully recreated by migrations alone, no manual SQL.
