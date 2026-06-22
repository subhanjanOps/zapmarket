# Plan — Dynamic Currency Management (`currency-service`)

**Status:** DRAFT — awaiting approval
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
- **NEW gRPC** (`currency-service`): `ListCurrencies`, `GetRates(base)`, `Convert(amountMinor, from, to)`.
- **REST via gateway** (unchanged path, enriched body):
  - `GET /api/v1/currencies` → `[{code,name,flag,enabled,decimals}]`
  - `GET /api/v1/currencies/rates` → `{base,as_of,stale,rates:{...}}` (superset of today's `{base,date,rates}`).

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
- `pkg/config/config.go` — service config + validation (`HTTP_PORT=8086`, `GRPC_PORT=50056`, `DB_NAME=currency`, `RATE_PROVIDER_URL`, `RATE_REFRESH_INTERVAL`, `REDIS_URL`).
- `domain/entities/currency.go`
- `domain/valueobjects/money.go`, `rate.go`, `currency_code.go`
- `domain/services/conversion_service.go` (+ `_test.go`)
- `domain/repositories/currency_repository.go`, `rates_repository.go` (interfaces — segregated)
- `application/ports/rates_provider.go`, `rates_cache.go`
- `application/dto/currency_dto.go`, `rates_dto.go`
- `application/usecases/list_currencies.go`, `get_rates.go`, `convert.go`, `ingest_rates.go` (+ `_test.go` each)
- `infrastructure/postgres/currency_repository.go`, `rates_repository.go`
- `infrastructure/redis/rates_cache.go`
- `infrastructure/external/frankfurter_provider.go`
- `interfaces/http/handler.go`, `interfaces/http/router.go`
- `interfaces/grpc/server.go`
- `interfaces/worker/rate_ingestor.go`
- `proto/currency.proto` + generated `proto/currencypb/*.go`
- `migrations/0001_currencies.sql`, `migrations/0002_exchange_rates.sql`, `migrations/0003_seed_currencies.sql` (seed the 25 currencies from the prototype list)
- `docs/` — swagger output via `pkg/swaggerx`
- `tests/unit/`, `tests/integration/`

### Files to modify
- `go.work` — add `use ./services/currency-service`.
- `services/api-gateway/main.go` — remove the inline `/api/v1/currencies/rates` closure that calls `internal/currency`; route `/api/v1/currencies*` to `currency-service` (static registry entry `currency-service` + DB route row). Delete `services/api-gateway/internal/currency/rates.go`.
- `services/seller-ui/lib/currency.tsx` — replace hardcoded `CURRENCIES` with a fetch of `GET /api/proxy/api/v1/currencies`; consume `as_of`/`stale` from the rates endpoint; keep localStorage cache + `detectCurrency`/`ZERO_DECIMAL` fallbacks.
- `services/seller-ui/app/api/fx-rates/route.ts` — point at gateway (or remove in favour of the existing `/api/proxy` path) so there is a single source of truth.
- `docker-compose.yml` / deployment config — register the `currency` DB and the new service (note: most infra is commented out today; add commented stanza consistent with repo style).
- `CLAUDE.md` — add `currency-service` to the services/ports table.

### New interfaces
- `RatesProvider` (`FetchLatest(ctx, base) (RateSet, error)`) — external FX abstraction (DIP; allows multi-provider later).
- `RatesRepository` (`UpsertLatest`, `LatestByBase`) and `CurrencyRepository` (`ListEnabled`, `Get`, `Upsert`) — segregated reader/writer where useful.
- `RatesCache` (`Get`, `Set`) — Redis abstraction.

### New DTOs
- `CurrencyDTO{Code,Name,Flag,Decimals,Enabled}`
- `RatesDTO{Base,AsOf,Stale,Rates map[string]float64}`
- `ConvertResultDTO{AmountMinor,Currency,RateUsed,AsOf}`

### New repositories
- `infrastructure/postgres.CurrencyRepository`, `infrastructure/postgres.RatesRepository` (raw `database/sql` + `lib/pq`, no ORM).
- `infrastructure/redis.RatesCache`.

### New use cases
- `ListCurrenciesUseCase` — enabled catalogue.
- `GetRatesUseCase` — cache→DB read, computes `stale` from `as_of` vs refresh interval.
- `ConvertUseCase` — pure conversion via `ConversionService`.
- `IngestRatesUseCase` — provider→DB→cache, called by the worker.

---

## Phase 4 — Implementation approach (Clean Arch / DDD / SOLID)
- **Thin handlers/consumers:** HTTP/gRPC/worker only parse, validate, call a use case, map response. No SQL, no provider calls, no business rules in interfaces.
- **Explicit DI:** constructor injection throughout; `main.go` is the only composition root. No globals/singletons.
- **Dependency rule:** domain imports nothing infra; use cases depend on ports/interfaces; infra implements them.
- **Pure domain:** `ConversionService` + value objects are framework-free and fully unit-testable (table-driven tests incl. zero-decimal currencies, missing rate, identity conversion).
- **Build order (TDD per `test-driven-development`):**
  1. Domain value objects + `ConversionService` (+ tests).
  2. Ports + use cases with fakes (+ tests).
  3. Infra adapters (postgres/redis/external).
  4. proto + gRPC server; HTTP handlers + swagger.
  5. Worker scheduler + `main.go` wiring + health/shutdown.
  6. Gateway rewire + delete `internal/currency`.
  7. UI consumption of dynamic list.

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

## Open decisions for you
1. **Port/DB name:** proposed HTTP `8086`, gRPC `50056`, DB `currency` — OK?
2. **Gateway → service for `/rates`:** route REST straight through (simplest) vs. gateway calls gRPC and re-serializes (preserves exact body control). Proposed: straight route, service owns the contract.
3. **Provider:** keep frankfurter.app as the only shipped provider for now? (interface allows adding more later.)
4. **Other UIs** (admin-ui, backoffice-ui) — rewire now or only seller-ui in this pass?
