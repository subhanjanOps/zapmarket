# Plan — Dynamic Currency Management (`currency-service`)

**Status:** APPROVED — ready for implementation
**Author:** Principal Engineer (Claude)
**Date:** 2026-06-21
**Scope decision:** Dedicated `currency-service` microservice (gRPC + REST), display/quote-oriented (no settlement changes).

---

## Phase 1 — Requirement Analysis

### Business goal
Let ZapMarket present prices in a buyer's local currency across all UIs, with an authoritative, centrally-managed catalogue of supported currencies and exchange rates. Replace the ad-hoc, hardcoded prototype (gateway `internal/currency` + hardcoded `CURRENCIES` list in `seller-ui`) with a single owning service so currency data is consistent, observable, and operable without code deploys.

### Scope (in)
- New `currency-service` owning: supported-currency catalogue, exchange-rate ingestion + storage, conversion logic.
- Scheduled rate ingestion from an external FX provider (frankfurter.app to start), with Redis caching and DB durability.
- gRPC API for internal callers: `ListCurrencies`, `GetRates`, `Convert`.
- REST API (via gateway) for UIs: `GET /api/v1/currencies`, `GET /api/v1/currencies/rates`.
- Gateway routes traffic to `currency-service` instead of its embedded `internal/currency` package.
- UIs consume the dynamic currency list instead of the hardcoded `CURRENCIES` array.

### Scope (out — explicitly deferred)
- Transactional/settlement multi-currency (orders/payments still settle in USD). No order/payment/catalog schema changes.
- Per-currency markup/margin pricing rules (catalogue supports the column but admin pricing UI is out).
- Multi-provider failover beyond a single configured provider (interface allows it; only one impl shipped).

### Risks
| Risk | Mitigation |
|---|---|
| External FX provider downtime | Serve last-good rates from DB/Redis; `stale` flag + `as_of` timestamp in responses. |
| Stale rates shown as fresh | Every response carries `as_of` + `stale` boolean; UI shows "rates as of …". |
| New service = new ops surface (port, DB, health) | Follow `service-template.md` exactly: health/readiness, metrics, graceful shutdown, config validation. |
| Float rounding errors | Store rates as `NUMERIC`; convert in minor units (cents); never settle on these values (display only). |
| Breaking the existing gateway endpoint contract | Keep response shape backward compatible (`{base,date,rates}`); add fields, don't remove. |

### Assumptions
- USD remains the base/storage currency (matches existing `fx:rates:USD` and stored-in-USD-cents model).
- frankfurter.app is acceptable as the first provider (already used by the prototype).
- Redis + PostgreSQL are available (gateway already depends on both).
- Inter-service sync calls use gRPC per architecture rules; UIs reach the service only through the gateway.

---

## Phase 2 — Architecture Design

### Services impacted
| Service | Change |
|---|---|
| **currency-service** (NEW, HTTP 8086 / gRPC 50056) | Owns currencies + rates; ingestion worker; gRPC + REST. |
| **api-gateway** | Remove `internal/currency`; add a route to `currency-service`; optionally call it via gRPC for the `/api/v1/currencies/rates` endpoint to preserve the public contract. |
| **seller-ui** (and other UIs later) | `lib/currency.tsx` fetches the supported-currency list + rates from the gateway instead of the hardcoded array. |

### APIs impacted
- **NEW gRPC** (`currency-service`): `ListCurrencies`, `GetRates(base)`. (`Convert` deferred — no caller yet, YAGNI.)
- **REST via gateway** (straight passthrough — service owns the contract):
  - `GET /api/v1/currencies` → `[{code,name,flag,decimals}]` (enabled only, filtered server-side; `auth_mode=none`)
  - `GET /api/v1/currencies/rates` → `{base,as_of,date,stale,rates:{...}}` (`date` kept as alias for `as_of` for backward compat; `auth_mode=none`)
  - `PUT /api/v1/admin/currencies/:code` → `{enabled:bool}` — **NEW** admin endpoint; JWT-protected (`role=admin`), accessible from backoffice UI

### Events impacted
None for this scope. (Future: `currency.rates.updated.v1` for cache invalidation fan-out — noted in Future Improvements.)

### Database changes (new `currency` DB, owned by the service)
```sql
-- supported currency catalogue (admin-managed)
CREATE TABLE currencies (
  code       CHAR(3)  PRIMARY KEY,          -- ISO 4217
  name       TEXT     NOT NULL,
  flag       TEXT     NOT NULL DEFAULT '',
  decimals   SMALLINT NOT NULL DEFAULT 2,   -- 0 for JPY/KRW/IDR…
  enabled    BOOLEAN  NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- latest rate per currency, base = USD (durable fallback for provider outages)
CREATE TABLE exchange_rates (
  base       CHAR(3) NOT NULL DEFAULT 'USD',
  quote      CHAR(3) NOT NULL REFERENCES currencies(code),
  rate       NUMERIC(18,8) NOT NULL,
  as_of      TIMESTAMPTZ NOT NULL,          -- provider's date
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (base, quote)
);
```
Redis cache key `fx:rates:USD` (reused) holds the marshalled latest snapshot with TTL = refresh interval.

### Sequence Diagram — buyer views a price
```
UI ──GET /api/v1/currencies/rates──▶ api-gateway ──gRPC GetRates(USD)──▶ currency-service
                                                                              │
                                              Redis HIT? ◀───────────────────┤ (cache)
                                                                              │ miss → DB latest
                                          ◀── {base,as_of,stale,rates} ───────┘
UI caches 1h (localStorage), converts USD-cents → local currency for display
```

### Sequence Diagram — scheduled rate ingestion (worker goroutine)
```
ticker(1h) ▶ IngestRatesUseCase
                 │ provider.FetchLatest(USD)  ──HTTP──▶ frankfurter.app
                 │ ratesRepo.UpsertLatest(rates, as_of)        ──▶ PostgreSQL
                 │ cache.Set("fx:rates:USD", snapshot, ttl)    ──▶ Redis
                 └ log.Info("rates ingested", count, as_of)
   on provider error: keep DB/Redis last-good, log warn, mark stale on next read
```

### Component Diagram
```
currency-service
├── interfaces/
│   ├── http/        REST handlers (thin)  ── pkg/httpx, pkg/swaggerx
│   ├── grpc/        CurrencyService server (thin)
│   └── worker/      rate ingestion scheduler (thin consumer-of-clock)
├── application/
│   ├── usecases/    ListCurrencies, GetRates, Convert, IngestRates
│   ├── dto/         RatesDTO, CurrencyDTO, ConvertResult
│   └── ports/       RatesProvider, RatesRepository, CurrencyRepository, RatesCache
├── domain/
│   ├── entities/    Currency
│   ├── valueobjects/ Money (minor units), Rate, CurrencyCode
│   └── services/    ConversionService (pure, table-driven, unit-tested)
└── infrastructure/
    ├── postgres/    currencyRepo, ratesRepo
    ├── redis/       ratesCache
    └── external/    frankfurterProvider (implements RatesProvider)
```

### Data flow
External provider → `IngestRatesUseCase` → (PostgreSQL durable + Redis cache). Reads: gateway/UI → `GetRates`/`Convert` use case → Redis (hit) or PostgreSQL (miss) → DTO. Conversion math is a pure domain service operating on minor units + `NUMERIC` rates.

---

## Phase 3 — Implementation Plan

### Files to create (`services/currency-service/`)
- `go.mod`, `.env.example`, `README.md`
- `cmd/main.go` — wiring, DI, health, graceful shutdown, worker start (mirrors `auth-service/cmd/main.go`).
- `pkg/config/config.go` — config + validation. Env vars: `HTTP_PORT=8086`, `GRPC_PORT=50056`, `DB_NAME=currency`, `RATE_PROVIDER_URL=https://api.frankfurter.app`, `RATE_REFRESH_INTERVAL=1h`, `MAX_RATE_AGE=24h`, `REDIS_URL`, `LOG_LEVEL=info`.
- `domain/entities/currency.go`
- `domain/valueobjects/money.go`, `rate.go`, `currency_code.go`
- `domain/services/conversion_service.go` (+ `_test.go`) — handles zero-decimal currencies (JPY, KRW, IDR from seed)
- `domain/repositories/currency_repository.go`, `rates_repository.go` (interfaces — segregated)
- `application/ports/rates_provider.go`, `rates_cache.go`
- `application/dto/currency_dto.go`, `rates_dto.go`
- `application/usecases/list_currencies.go` — returns only `enabled=true` rows
- `application/usecases/get_rates.go` — staleness = `now() − as_of > RATE_REFRESH_INTERVAL × 2`; returns 503 if `now() − as_of > MAX_RATE_AGE`
- `application/usecases/ingest_rates.go` — triggered by worker tick (no immediate fetch on startup)
- `application/usecases/toggle_currency.go` — **NEW** enable/disable a currency (admin use case)
- (+ `_test.go` for each use case)
- `infrastructure/postgres/currency_repository.go`, `rates_repository.go`
- `infrastructure/redis/rates_cache.go` — key `currency:rates:USD` (new namespace; shared Redis instance)
- `infrastructure/external/frankfurter_provider.go`
- `interfaces/http/handler.go` — public handlers (`ListCurrencies`, `GetRates`) + **NEW** admin handler (`ToggleCurrency`, JWT `role=admin`)
- `interfaces/http/router.go`
- `interfaces/grpc/server.go` — `ListCurrencies`, `GetRates` RPCs (`Convert` deferred)
- `interfaces/worker/rate_ingestor.go`
- `proto/currency.proto` + generated `proto/currencypb/*.go` — flat `currency` proto package (consistent with existing protos)
- `migrations/0001_currencies.sql`, `migrations/0002_exchange_rates.sql`
- `migrations/0003_seed_currencies.sql` — seed all 25 currencies from `seller-ui/lib/currency.tsx` with `decimals` values: JPY=0, KRW=0, IDR=0, all others=2
- `docs/` — swagger output via `pkg/swaggerx` at `/v1/docs/swagger.json`
- `tests/unit/`, `tests/integration/` — use existing `docker-compose.yml` for integration test infra; coverage targets advisory (domain ≥90%, use cases ≥85%)

### Files to modify
- `go.work` — add `use ./services/currency-service`.
- `services/api-gateway/main.go` — remove inline `/api/v1/currencies/rates` closure; add straight-passthrough routes for `GET /api/v1/currencies`, `GET /api/v1/currencies/rates` (`auth_mode=none`) and `PUT /api/v1/admin/currencies/:code` (`auth_mode=admin`) pointing to `currency-service`. Delete `services/api-gateway/internal/currency/rates.go`.
- `services/seller-ui/lib/currency.tsx` — extend `CurrencyProvider` to also fetch `GET /api/proxy/api/v1/currencies` for the dynamic list (replaces hardcoded `CURRENCIES`); update `fetchRates` to consume `as_of`/`stale`; keep `ratesDate` as "rates as of …" footnote (subtitle, not banner); keep existing localStorage 1h cache + `detectCurrency`/`ZERO_DECIMAL` fallbacks.
- `services/seller-ui/app/api/fx-rates/route.ts` — **delete** (consumers already use `/api/proxy/api/v1/currencies/rates`; single source of truth through gateway).
- `docker-compose.yml` — add **live (uncommented)** `currency-service` container + `currency` PostgreSQL DB stanza.
- `CLAUDE.md` — add `currency-service | 8086 | 50056 | currency` to the services/ports table.

### New interfaces
- `RatesProvider` (`FetchLatest(ctx, base) (RateSet, error)`) — external FX abstraction (DIP; allows multi-provider later).
- `RatesRepository` (`UpsertLatest`, `LatestByBase`) and `CurrencyRepository` (`ListEnabled`, `Get`, `Toggle`) — segregated.
- `RatesCache` (`Get`, `Set`) — Redis abstraction; key `currency:rates:USD`.

### New DTOs
- `CurrencyDTO{Code,Name,Flag,Decimals}` — no `Enabled` field (server filters; public callers only see enabled)
- `RatesDTO{Base,AsOf,Date,Stale,Rates map[string]float64}` — `Date` is alias for `AsOf` (backward compat)

### New repositories
- `infrastructure/postgres.CurrencyRepository`, `infrastructure/postgres.RatesRepository` (raw `database/sql` + `lib/pq`, no ORM).
- `infrastructure/redis.RatesCache`.

### New use cases
- `ListCurrenciesUseCase` — returns only `enabled=true` rows.
- `GetRatesUseCase` — Redis→DB read; `stale = now()−as_of > interval×2`; HTTP 503 if `now()−as_of > MAX_RATE_AGE`.
- `IngestRatesUseCase` — provider→DB→cache, called by the worker tick.
- `ToggleCurrencyUseCase` — **NEW** enable/disable a currency by code; requires `admin` role (enforced at handler, not use case).

---

## Phase 4 — Implementation approach (Clean Arch / DDD / SOLID)
- **Thin handlers/consumers:** HTTP/gRPC/worker only parse, validate, call a use case, map response. No SQL, no provider calls, no business rules in interfaces.
- **Explicit DI:** constructor injection throughout; `main.go` is the only composition root. No globals/singletons.
- **Dependency rule:** domain imports nothing infra; use cases depend on ports/interfaces; infra implements them.
- **Pure domain:** `ConversionService` + value objects are framework-free and fully unit-testable (table-driven tests incl. zero-decimal currencies, missing rate, identity conversion).
- **Build order (TDD per `test-driven-development`):**
  1. Domain value objects + `ConversionService` (+ tests; covers zero-decimal JPY/KRW/IDR).
  2. Ports + use cases (`ListCurrencies`, `GetRates`, `IngestRates`, `ToggleCurrency`) with fakes (+ tests).
  3. Infra adapters (postgres/redis/external).
  4. proto (`currency` package) + gRPC server (`ListCurrencies`, `GetRates`); HTTP handlers (public + admin) + swagger.
  5. Prometheus metrics (`/metrics`): ingestion count, fetch latency histogram, cache hit counter.
  6. Worker scheduler + `main.go` wiring + health/shutdown + auto-migrations.
  7. Gateway rewire (straight passthrough; admin route JWT `role=admin`) + delete `internal/currency` + update `docker-compose.yml`.
  8. seller-ui: dynamic list in `CurrencyProvider` + delete `fx-rates/route.ts` + stale footnote.

---

## Phase 5 — Validation checklist
- **Architecture violations:** verify no infra import in domain/application; handlers contain no SQL/HTTP-client calls.
- **Security:** provider URL from config (no hardcoded secrets); REST endpoints are public read-only (rates/list) — confirm gateway `auth_mode=none` only for GET; no PII logged.
- **Performance:** Redis-first reads; single scheduled provider call per interval (not per request); `NUMERIC` math; response `Cache-Control` preserved.
- **Testing gaps:** domain ≥90%, use cases ≥85%; integration test for ingestion against a stubbed provider; contract test for the gateway REST shape (must stay a superset of `{base,date,rates}`).
- **Standards:** swagger via `pkg/swaggerx` at `/v1/docs/swagger.json` (per project memory); status strings N/A here; structured logging only.

---

## Final Output

### Architecture summary
A new clean-architecture `currency-service` becomes the single owner of supported currencies and exchange rates. A worker ingests rates on a schedule into PostgreSQL (durable) + Redis (fast path); use cases serve reads via gRPC (internal) and REST (UIs, through the gateway). The gateway stops embedding currency logic and simply routes; UIs stop hardcoding the currency list. Money is still stored and settled in USD — this layer is display/quote only.

### Implementation summary
New service under `services/currency-service` following `service-template.md`; `go.work` + gateway + `seller-ui` modified; `internal/currency` deleted; new `currency` DB with `currencies` + `exchange_rates`; backward-compatible REST contract.

### Risks
Provider outage (mitigated by durable last-good + `stale` flag), new ops surface (mitigated by template compliance), float rounding (mitigated by `NUMERIC` + minor-unit math + display-only scope).

### Future improvements
- `currency.rates.updated.v1` Kafka event for cross-service cache invalidation.
- Multi-provider failover + provider health scoring.
- Admin CRUD UI for enabling currencies / markup rules.
- Promote to transactional multi-currency (rate snapshot at checkout) if/when settlement in local currency is required.

---

## Resolved Decisions

All decisions are locked. Items marked *(chosen)* were selected by the engineering lead; items marked *(default)* were resolved by the plan author where the preference was deferred.

### A — Infrastructure & Ports
| # | Decision | Resolution |
|---|---|---|
| 1 | Port/DB assignment | HTTP `8086`, gRPC `50056`, DB `currency` ✓ |
| 2 | Docker Compose stanza | **Live (uncommented)** — `currency-service` + `currency` DB added as active services |
| 3 | Redis | *(default)* **Shared** Redis instance with gateway. New key namespace `currency:rates:USD` avoids collision with gateway's existing `fx:rates:USD` key |
| 4 | Migrations | **Auto-run at startup** (same as auth-service) |

### B — Rate Ingestion & Staleness
| # | Decision | Resolution |
|---|---|---|
| 5 | Startup fetch | **Wait for first tick** — no immediate fetch on startup |
| 6 | Staleness threshold | *(default)* `stale = now() − as_of > RATE_REFRESH_INTERVAL × 2` — no separate env var |
| 7 | Provider error behaviour | **(b)** — serve last-good with `stale: true`; return **HTTP 503** if `now() − as_of > MAX_RATE_AGE` (env var, default `24h`) |
| 8 | Provider URL | *(default)* `RATE_PROVIDER_URL=https://api.frankfurter.app`; no separate provider-name config (YAGNI) |

### C — Gateway Routing
| # | Decision | Resolution |
|---|---|---|
| 9 | Routing strategy | **Straight passthrough** — service owns the response contract |
| 10 | Auth mode | `auth_mode=none` for GET currency routes ✓ |
| 11 | Backward compat | *(default)* **Keep `date` as alias** for `as_of` in JSON response — both fields returned, zero cost, no consumer breakage |

### D — Currency Catalogue
| # | Decision | Resolution |
|---|---|---|
| 12 | Seed source | **Seed migration is authoritative** — 25 currencies from `seller-ui/lib/currency.tsx` with `decimals` added |
| 13 | Zero-decimal currencies | **JPY=0, KRW=0, IDR=0** (the three present in the seed list); `ZERO_DECIMAL` set in seller-ui expanded to match |
| 14 | Admin catalogue | **Admin endpoint included in scope**: `PUT /api/v1/admin/currencies/:code` `{enabled:bool}`, JWT `role=admin`, accessible from backoffice UI |

### E — API & Contracts
| # | Decision | Resolution |
|---|---|---|
| 15 | Proto package | *(default)* **Flat `currency` package** (`proto/currency.proto`) — consistent with existing protos |
| 16 | `Convert` RPC | *(default)* **Deferred** — no internal caller exists (YAGNI); interface is designed for it but not implemented |
| 17 | REST response shape | *(default)* **Filter server-side** — `GET /api/v1/currencies` returns only `enabled=true` rows; no `enabled` field in response |

### F — Testing & Observability
| # | Decision | Resolution |
|---|---|---|
| 18 | Integration test infra | *(default)* **Existing `docker-compose.yml`** — no dedicated test compose yet |
| 19 | Coverage floor | *(default)* **Advisory only** — domain ≥90%, use cases ≥85%, not CI-enforced |
| 20 | Prometheus metrics | **Yes — all three**: ingestion success/failure count, rate fetch latency (histogram), cache hit rate (counter) |
| 21 | Log level | *(default)* `LOG_LEVEL=info` prod / `debug` dev; ingestor logs individual rate values at DEBUG |

### G — UI & Seller-UI
| # | Decision | Resolution |
|---|---|---|
| 22 | Currency list fetch strategy | *(default)* **React context/provider** — extend existing `CurrencyProvider` to also fetch dynamic list; 24h localStorage cache |
| 23 | `fx-rates/route.ts` fate | *(default)* **Delete** — consumers already use `/api/proxy/api/v1/currencies/rates`; single source of truth |
| 24 | Other UIs scope | *(default)* **seller-ui only** in this pass |
| 25 | Stale indicator | *(default)* **Subtle timestamp footnote** ("rates as of [date/time]") — uses existing `ratesDate` state already in `CurrencyProvider` |
