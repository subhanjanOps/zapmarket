# currency-service Review

## Executive Summary

`currency-service` is a well-structured implementation that respects the repository's clean-architecture conventions far better than most first-pass services. The four-layer split (`domain`/`application`/`infrastructure`/`interfaces`) is real and the dependency direction is correct: domain has zero infrastructure imports, the application layer depends only on ports and domain repository interfaces, and infrastructure implements those contracts. Ports/adapters are used idiomatically (`RatesProvider`, `RatesCache`, `EventPublisher`), the fallback provider is a clean Strategy/Decorator, and Kafka is correctly optional with a no-op default. Use-case tests with fakes exist and cover the important branches (cache hit, stale, ingest publish/no-publish, fallback paths).

The issues are mostly correctness-at-the-edges and security, not architecture. The most important problems: (1) the admin toggle authorizes purely on a spoofable `X-User-Role` header with no defense-in-depth, (2) the background worker shares the root context but is not awaited on shutdown, so an in-flight ingest is abandoned rather than drained, (3) the rate-provider URL is taken from config and never validated (SSRF / mis-config surface), and (4) cache invalidation on the admin toggle is missing, so a disabled currency keeps being served from the rates cache. None are architectural rewrites; all are bounded fixes.

**Verdict: APPROVED WITH RECOMMENDATIONS.**

---

## Critical Findings

None that block merge outright, but the two security items below are close to the line and should be resolved before this service is exposed to untrusted gateway traffic.

---

## High Priority Findings

### H1 — Admin authorization trusts a spoofable header with no defense-in-depth
`interfaces/http/handler.go` `ToggleCurrency` authorizes solely on `r.Header.Get("X-User-Role") == "admin"`. This is acceptable *only* if the gateway unconditionally strips client-supplied `X-User-Role` on every inbound request and re-injects it after JWT validation. That guarantee lives entirely in the gateway and is invisible from this service. If the gateway is ever misconfigured, the route is reachable directly (it listens on `:8086`), or `strip_prefix`/auth_mode is wrong for this path, any caller can send `X-User-Role: admin` and toggle currencies.

Recommendations:
- Treat the header as a hint, not a credential. At minimum, document the exact gateway contract (header is stripped + re-set) in a comment with a pointer to the gateway route config, and add a startup log asserting the service must not be exposed publicly.
- Stronger: validate the bearer token via the auth-service `ValidateToken` gRPC like other services do (the monorepo already has this pattern in `product-catalog-service/internal/middleware`), rather than trusting a plaintext header. This is the established RBAC mechanism in ZapMarket; the currency-service is the outlier.

### H2 — Rate provider URL is never validated
`cmd/main.go` passes `cfg.RateProviderURL` straight into `NewFrankfurterProvider`, and the secondary provider URL is hardcoded. The providers then `fmt.Sprintf("%s/latest?from=%s", baseURL, base)` with no parsing. A malformed or attacker-influenced `RATE_PROVIDER_URL` (it is env-driven) produces a silent mis-fetch and is an SSRF vector if that env ever derives from untrusted input. Validate at config load: `url.Parse`, require `https` scheme and a non-empty host, reject on failure in `config.Load()`. This is cheap and turns a silent runtime failure into a fail-fast at boot.

### H3 — Worker is not drained on shutdown
`cmd/main.go` launches `go ingestor.Run(ctx)` but never waits for it. On SIGTERM, `cancel()` is called, then HTTP `Shutdown` and `grpcSrv.GracefulStop()` run, and `main` returns — the process exits while an in-flight `ingest.Execute` (an HTTP fetch + multi-statement Postgres transaction) may still be running on the cancelled context. The DB transaction will roll back (fine) but the shutdown sequence makes no attempt to let it finish cleanly. Add a `sync.WaitGroup` (or have `Run` return and be awaited) so shutdown drains the worker within the existing 15s window. Architecture-principles "Reliability" and resource cleanup both point here.

### H4 — Admin toggle does not invalidate the rates cache (stale serving)
`ToggleCurrencyUseCase` updates the `currencies` table but the rates cache (`currency:rates:USD`) is keyed by base, not by currency enablement, and is independent. More importantly, `GetRatesUseCase` and `ListCurrenciesUseCase` are wholly separate paths — disabling a currency removes it from `ListEnabled` but `GetRates` still returns its rate from cache/DB, because rates are never filtered by enabled set. If the product intent is "disabled currency must disappear from quotes," this is a functional gap. Decide the contract explicitly: either (a) `GetRates` joins against enabled currencies, or (b) document that rates are intentionally returned for all ISO codes regardless of catalogue toggle. Right now it is implicit.

---

## Medium Priority Findings

### M1 — `IngestRatesUseCase.Execute` does two jobs and silently swallows marshal errors
The use case is ~50 lines and within limits, but it builds the persistence rows, then *re-builds* a second `rateMap` for the cache from `rateSet.Rates`, duplicating the base/quote handling already done for `rows`. Extract a small `buildRateMap` helper so the cache payload and the persisted rows derive from one source of truth. Also `payload, _ := json.Marshal(...)` discards the error (violates engineering-standards "Never ignore errors"); marshaling a `map[string]interface{}` of primitives won't fail in practice, but the standard is explicit — at least log it.

### M2 — `GetRates` cache miss vs. cache error are conflated
`if cached, ok, err := uc.cache.Get(...); err == nil && ok` correctly falls through to DB on a cache error, which is the right availability tradeoff — but the cache error is then completely discarded. A persistently failing Redis (every request going to Postgres) would be invisible. Log the cache error at warn level before falling through so the degradation is observable. Observability is a stated requirement.

### M3 — `as_of` precision differs between providers, undermining history idempotency
Frankfurter parses a date (`2006-01-02`, midnight) while OpenExchangeRates uses `time.Now().UTC().Truncate(24*time.Hour)`. The history table's idempotency key is `(base, quote, as_of)`. The two providers can produce different `as_of` values for "the same day" if Frankfurter's published date lags the wall clock (common — Frankfurter publishes on a delay). After a fallback day, history may carry two rows for what is logically one day. Normalize `as_of` to a date at the use-case boundary so idempotency is provider-independent.

### M4 — `LatestByBase` returns `latestAsOf` derived from row scan, but staleness is judged on it
`GetRatesUseCase` computes `age := time.Since(asOf)` from the max `as_of` across rows. Because `as_of` is a date (midnight UTC), a rate fetched at 23:00 today already reads as ~23h old, and tomorrow morning it crosses the 24h `MaxRateAge` and starts returning 503 even though the worker is healthy and the rate is the latest published value. Staleness should be measured against `fetched_at` (when the worker last successfully stored), not the provider's nominal date. This is a likely production false-positive 503.

### M5 — gRPC `GetRates` reports `codes.Unavailable` for all errors
`server.go` maps every `getRates.Execute` error to `codes.Unavailable`. An internal DB scan failure is not "unavailable" to a client in the retry sense. Distinguish `ErrRatesTooStale`/no-rows (Unavailable / FailedPrecondition) from genuine internal errors (Internal), mirroring the HTTP handler which already does this with `errors.As`.

### M6 — Dead code / misleading comment in gRPC server
The trailing `var _ = json.Marshal // ensure encoding/json is imported` plus the "jsonCodec / RegisterJSONCodec" comment describe a mechanism that does not exist in this file. `encoding/json` is unused. Remove the import and the comment block — it will confuse the next reader and contradicts coding-guidelines on comments.

---

## Low Priority Findings

- **L1** — `cmd/main.go` import alias `nethhttp` is a typo-ish alias for `net/http`; `stdhttp` or just aliasing the local handler package would read better. Cosmetic.
- **L2** — `GetRates` HTTP/gRPC hardcode base `"USD"` (HTTP) while gRPC accepts a `base` param defaulting to USD. The HTTP endpoint ignores any base entirely. If multi-base is a non-goal, document it; if it's a goal, the HTTP handler silently drops the capability.
- **L3** — `UpsertLatest` runs two sequential per-row loops issuing 2N round-trips inside one transaction. For ~30–160 currencies this is fine, but a multi-row `INSERT ... VALUES` (or `pq.CopyIn`) would cut latency materially if the currency set grows. Note for scale, not urgent.
- **L4** — `metrics.RecordIngest` is the only worker metric; there is no gauge for "seconds since last successful ingest," which is the single most useful alert for this service. Add it so staleness is alertable before the 503s start (ties to M4).
- **L5** — `ErrRatesTooStale.Error()` is surfaced verbatim into the HTTP body (`"exchange rates unavailable: ..."`). Low risk (no secrets), but leaking internal age phrasing to clients is unnecessary; a generic 503 message would do.
- **L6** — No test exercises the HTTP handler layer (role rejection, bad date parse, stale→503 mapping) or the gRPC server. Use-case coverage is good; the interface layer's parsing/validation/error-mapping is untested.
- **L7** — `conversion_service.go` `zeroDecimal` map is hardcoded to three currencies; real ISO 4217 has more zero-decimal currencies (e.g. CLP, VND, XOF). If conversion is user-facing, this will round incorrectly. Source it from the `currencies.decimals` column instead of a hardcoded map.

---

## Recommended Refactoring Plan

Ordered by value/effort:

1. **Security first (H1, H2):** Validate `RateProviderURL` in `config.Load()` (fail-fast). Decide and document the admin-auth contract; ideally swap the `X-User-Role` check for the existing auth-service gRPC validation pattern used elsewhere in the monorepo.
2. **Shutdown correctness (H3):** Add a `WaitGroup` around the worker goroutine and wait on it inside the 15s shutdown window.
3. **Staleness semantics (M4, M3, L4):** Measure staleness against `fetched_at`, normalize `as_of` to a date at the use-case boundary, and add a "seconds since last successful ingest" gauge. These three are one coherent change to how time is handled.
4. **Cache/catalogue contract (H4):** Decide whether disabled currencies must vanish from `GetRates`; implement the join or document the intentional decoupling.
5. **Cleanups (M1, M2, M5, M6):** Extract `buildRateMap`, log cache-get errors, refine gRPC status codes, delete the dead json codec stub.
6. **Test gaps (L6):** Add httptest-based handler tests for the admin-role rejection and date validation, plus a gRPC server test.

---

## Final Verdict

**APPROVED WITH RECOMMENDATIONS.**

The architecture is sound and idiomatic — clean layering, correct dependency direction, good port/adapter use, optional Kafka, and a real test suite with fakes. Merge is acceptable, but H1–H4 (admin-auth hardening, provider-URL validation, worker drain on shutdown, and the cache/staleness semantics in M3/M4) should be addressed before this service handles production gateway traffic, as they are reliability- and security-affecting rather than cosmetic.
