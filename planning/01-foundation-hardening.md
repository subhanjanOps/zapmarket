# Stage 1 — Foundation Hardening

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
- [ ] Grep both live services for any service-local `config` or `crypto` package that
      duplicates `pkg/config` / `pkg/crypto` and delete the duplicate, repointing imports.
- [ ] Confirm `auth-service` and `product-catalog-service` both import `pkg/database`,
      `pkg/errors`, `pkg/grpcx`, `pkg/httpx`, `pkg/logger` rather than ad-hoc equivalents.
- [ ] Run `go work sync` and `go build ./...` from repo root after the sweep.

### 1.2 Domain contract cleanup (Phase 4)
- [ ] `services/product-catalog-service/internal/service/repository_interfaces.go` currently
      lives in the `service` package. Move the interfaces into a new
      `internal/domain/contracts` package (or `internal/repository/contracts.go` if that
      reads more naturally with the existing layout — pick one and apply it consistently).
- [ ] Confirm `internal/service/*.go` depends only on the interface types, not on concrete
      `internal/repository` structs (no `repository.NewProductRepository` calls inside
      `service` — those should be wired in `main.go`).
- [ ] Repeat the same audit for `auth-service` — its `internal/service` likely has the same
      interface-in-service-package pattern; give it the same contracts location.
- [ ] This is the template the four new services (Stages 3-6) must follow from day one —
      write contracts in `internal/domain/contracts` before writing the repository struct.

### 1.3 Database migrations (Phase 6)
- [ ] Add `golang-migrate` (`github.com/golang-migrate/migrate/v4`) as a dependency in
      `pkg/database` or a new `pkg/migrate` helper — decide based on whether migration
      logic needs DB-pool internals or just a DSN string (likely just a DSN, so a thin
      `pkg/migrate` wrapper is cleaner).
- [ ] Create `migrations/` directories per service (e.g.
      `services/auth-service/migrations/0001_init.up.sql` /
      `.down.sql`) capturing the **current** schema as migration 0001, since these tables
      already exist from manual setup — this makes migrations the source of truth going
      forward instead of a parallel undocumented schema.
- [ ] Add a `migrate` subcommand or a small `cmd/migrate/main.go` per service that runs
      pending up migrations against `DB_HOST`/`DB_PORT`/etc. from env.
- [ ] Wire migration execution into `docker-entrypoint-initdb.d/` or into each service's
      startup path (decide: run-on-boot vs. explicit `go run ./cmd/migrate` step in
      docker-compose `depends_on` — run-on-boot is simpler for local dev, explicit step is
      safer for prod; recommend run-on-boot guarded by an env flag).

## Out of scope
- New service scaffolding (Stages 3-6).
- Outbox table — that's introduced in Stage 5 alongside Order Management, using the
  migration system built here.

## Definition of done
- `go build ./...` and `go vet ./...` pass clean from repo root.
- No `internal/service` package in either live service defines a repository interface.
- Both services can be torn down (`docker compose down -v`) and brought back up with
  schema fully recreated by migrations alone, no manual SQL.
