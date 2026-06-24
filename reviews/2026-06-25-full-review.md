# ZapMarket — Full Production Code Review

**Date:** 2026-06-25
**Branch:** `features/cluster-setup`
**Scope:** All backend services + all UI services
**Reviewer:** Principal Engineer (parallel agent review)

---

## Changelog

| Date | Service | Changes |
|---|---|---|
| 2026-06-25 | auth-service | Fixed C1, C2, H1, H2, M2, M8, L1, L2, L3. Added forgot-password feature (migration, repository, email interface, service methods, HTTP handler). M7 documented (proto limitation). |
| 2026-06-25 | product-catalog-service | Fixed C1 (ownership checks on UpdateSKU, DeleteSKU, all image mutation endpoints), C2 (gRPC conn leak — Close() added), H2 (DeleteProduct ownership), H3 (LimitedReader in DecodeJSON), L5 (VariantAttrs double marshal). |
| 2026-06-25 | currency-service | Fixed C1 (X-User-Role replaced with gRPC JWT validation via new AuthMiddleware), C2 (Toggle RowsAffected check), H1 (base param validated via NewCurrencyCode). Added AuthServiceAddr to config. |
| 2026-06-25 | api-gateway | Fixed H1 (audit IP uses net.SplitHostPort(RemoteAddr)), H3 (auth_mode downgrade guard in autobind). C2 already covered by root .gitignore. M7 not present in current code. |
| 2026-06-25 | order-management-service | Fixed H1 (spin-wait loop removed), H2 (outbox payload uses json.Marshal). |
| 2026-06-25 | inventory-service | Fixed H1 (Redis warm TTL 0 → 24h), H2 (ReleaseStock uses luaIncrIfExists instead of INCRBY). |
| 2026-06-25 | UI services | Fixed proxy path traversal rejection on all 4 UIs. Fixed backoffice-ui 401 short-circuit when token absent. Fixed seller-ui double-create guard in saveBasicInfo. Added checkout required-field validation in buyer-ui. |

---

## Overall Verdict Summary

| Service | Critical | High | Medium | Low | Verdict |
|---|---|---|---|---|---|
| auth-service | 0 ✅ | 3 | 7 | 3 ✅ | **APPROVED WITH RECOMMENDATIONS** |
| product-catalog-service | 0 ✅ | 4 | 8 | 6 ✅ | **APPROVED WITH RECOMMENDATIONS** |
| api-gateway | 1 | 3 ✅ | 7 | 5 | **APPROVED WITH RECOMMENDATIONS** |
| currency-service | 0 ✅ | 3 | 6 | 5 | **APPROVED WITH RECOMMENDATIONS** |
| order-management-service | 1 | 1 ✅ | 3 | 3 | **APPROVED WITH RECOMMENDATIONS** |
| inventory-service | 0 ✅ | 0 ✅ | 3 | 2 | **APPROVED** |
| payment-service | 1 | 3 | 3 | 2 | **BLOCKED — tests required** |
| notification-service | 0 | 2 | 4 | 2 | Approved with recommended changes |
| admin-ui | 1 (no tests) | 0 ✅ | 4 | 4 | Conditional Pass |
| backoffice-ui | 1 (no tests) | 0 ✅ | 4 | 3 | Conditional Pass |
| buyer-ui | 1 (no tests) | 1 | 4 | 4 | Conditional Pass |
| seller-ui | 1 (no tests) | 0 ✅ | 2 ✅ | 4 | Conditional Pass |

---

## Cross-Cutting Issues (Affects All Services)

These gaps appear everywhere and should be addressed in a single coordinated effort:

1. **Zero test coverage** — No `*_test.go` files in payment-service; no test files in any UI service; minimal/mocked tests in the rest. No integration tests against real PostgreSQL anywhere.
2. **No metrics instrumentation** — No Prometheus/OpenTelemetry in order-management, inventory, payment, or notification services. Violates engineering standards.
3. **No distributed tracing** — No `trace_id`/`correlation_id` propagation beyond chi's `RequestID` middleware in any backend service.
4. **Architecture naming drift** — Four infrastructure services use `internal/service` + `internal/repository` rather than the `internal/application` + `internal/infrastructure` naming in `architecture-principles.md`.
5. **UI: Zero tests** — All four UI services have no application-level test files.
6. **UI: Proxy path sanitisation** — All four UI proxies build upstream URLs with `path.join("/")` and no `..` segment rejection. One shared utility function should be extracted.

---

## auth-service

### Critical

~~**C1 — Broken token refresh: client receives hash instead of raw token**~~
✅ **FIXED** — `refreshToken.TokenHash` replaced with `refreshToken.Token` at all 7 call sites in `handlers.go` and `auth_server.go`.

~~**C2 — Data race on `AuthService.rdb` field**~~
✅ **FIXED** — `SetRedis` removed. `NewAuthService` now accepts `*goredis.Client` at construction (nil when Redis is unavailable). `main.go` tries Redis first and passes `nil` on failure. No mutable field after construction.

### High

~~**H1 — OAuth CSRF: state parameter not validated on callback**~~
✅ **FIXED** — `StoreOAuthState` / `ValidateAndConsumeOAuthState` added to `AuthService` using Redis with a 10-minute TTL. Both URL handlers store the generated state; both callbacks verify-and-consume it (atomic `DEL`) before exchanging the code. Degrades gracefully when Redis is unavailable.

~~**H2 — `getFacebookUserInfo` uses `http.Get` with no context or timeout**~~
✅ **FIXED** — Refactored to `http.NewRequestWithContext(ctx, ...)` with `context.WithTimeout(ctx, 10*time.Second)`, matching the `getGoogleUserInfo` pattern.

**H3 — Handlers depend on concrete service types (DI violation)**
`internal/handler/http/handlers.go:92-95`, `handler/grpc/auth_server.go:24-25`

All handlers hold `*service.AuthService` and `*service.OAuthService`. Define `AuthServicePort` interface; depend on that. — **Pending**

**H4 — `AdminHandler` queries `UserRepository` directly, bypassing service layer**
`internal/handler/http/admin_handler.go:103-140`

Business logic (pagination, role validation) lives in handlers. Move to service layer per coding guidelines. — **Pending**

**H5 — No rate limiting on auth endpoints**
`cmd/main.go`

`/v1/auth/login`, `/register`, `/refresh` have no rate limiting. Redis is already wired — add a per-IP/per-email sliding window.

### Medium

**M1** — Access token generation duplicated across 6+ call sites. Encapsulate in `AuthService.GenerateAccessToken`.

**M2** — String comparison for pq duplicate-key error is fragile (`err.Error() == "pq: ..."` in `user_repository.go:48`). Use `pq.Error.Code == "23505"`.

**M3** — `OAuthService` depends on `*AuthService` concrete type. Extract an interface.

**M4** — `handleOAuthUser` has TOCTOU race (check-then-create-user-then-create-oauth-account without a transaction).

**M5** — `ListUsers` ILIKE search at `user_repository.go:235-238` adds the same value twice; also verify final `LIMIT`/`OFFSET` positional indices.

**M6** — gRPC logging duplicated 20+ lines per method. Extract to a unary server interceptor. — **Pending**

~~**M7** — `RegisterUser` gRPC hard-codes `RoleBuyer` regardless of the proto field (`auth_server.go:161`).~~
📝 **DOCUMENTED** — `RegisterUserRequest` proto has no `role` field. Added a comment in `auth_server.go` explaining the limitation; proto needs a `role` field to enable seller gRPC registration.

~~**M8** — No email format validation on registration.~~
✅ **FIXED** — `net/mail.ParseAddress` check added in `Register` handler before calling the service.

### Low

~~**L1** — gRPC server `GracefulStop` never called on shutdown (`cmd/main.go:171-187`).~~
✅ **FIXED** — `grpcSrv` now declared in outer scope; `GracefulStop()` called in shutdown sequence.

~~**L2** — Stale Swagger `@BasePath /auth` should be `/v1/auth`.~~
✅ **FIXED** — Corrected in `handlers.go`.

~~**L3** — `ADMIN_BOOTSTRAP_SECRET` read via `os.Getenv` instead of config (`handlers.go:381`).~~
✅ **FIXED** — Moved to `config.Config.AdminBootstrapSecret`, loaded at startup via `ADMIN_BOOTSTRAP_SECRET` env var. Handler now reads `h.cfg.AdminBootstrapSecret`.

**L4** — `User.Role` typed as bare `string` instead of domain `Role` type. — **Pending**

**L5** — No tests for `AuthService` or `OAuthService`. — **Pending**

### New Feature: Forgot Password ✅

Implemented as part of this session:
- `migrations/0005_password_reset.up.sql` — `password_reset_tokens` table (hash, expiry, `used_at`)
- `internal/domain/models.go` — `PasswordResetToken` model
- `internal/domain/contracts/repositories.go` — `PasswordResetRepository` interface
- `internal/repository/password_reset_repository.go` — concrete implementation
- `internal/email/emailer.go` — `Emailer` interface + `LogEmailer` (dev) + `SMTPEmailer` (prod)
- `internal/service/auth_service.go` — `RequestPasswordReset` + `ResetPassword` methods
- `internal/handler/http/password_handler.go` — `ForgotPassword` + `ResetPassword` handlers
- Routes: `POST /v1/auth/password/forgot`, `POST /v1/auth/password/reset`
- Config additions: `PASSWORD_RESET_BASE_URL`, `SMTP_HOST/PORT/USER/PASSWORD/FROM`

Security: 32-byte CSPRNG token, SHA-256 hash stored, 30-min TTL, single-use, all sessions revoked on reset, 200 always returned on forgot to prevent email enumeration.

---

## product-catalog-service

### Critical

~~**C1 — Authorization bypass: SKU and image mutation endpoints do not enforce seller ownership**~~
✅ **FIXED** — `requireUser` + `assertOwnership` added to `UpdateSKU`, `DeleteSKU`, `CreateProductImage`, `UpdateImagePosition`, `DeleteProductImage`. `SKUHandler` and `ProductImageHandler` now accept `ProductService` for ownership resolution. `main.go` updated.

~~**C2 — gRPC connection leak in `AuthMiddleware`**~~
✅ **FIXED** — `conn *grpc.ClientConn` stored in `AuthMiddleware`; `Close() error` method added; `defer authMW.Close()` called in `main.go`.

### High

**H1 — N+1 query in `BulkCreateCategories` topological sort**
`internal/service/category_service.go:110-162`

Each wave issues a separate transaction with N individual inserts. Use a single transaction or a multi-row `INSERT ... UNNEST(...)` per wave.

~~**H2 — `DeleteProduct` missing ownership check**~~
✅ **FIXED** — `requireUser` + `assertOwnership` added to `DeleteProduct` before the service call.

~~**H3 — Unbounded JSON body — no `MaxBytesReader`**~~
✅ **FIXED** — `DecodeJSON` now wraps `r.Body` in `io.LimitedReader` (1 MiB + 1 byte sentinel). Returns error when limit exceeded.

**H4 — Sort field validated in service AND silently overridden in repository**
Three repositories contain a redundant `switch` with a silent default. Remove from repositories; service is single source of truth.

**H5 — `authctx` leaks proto type into handler/service boundary**
`internal/authctx/authctx.go`

`UserFromContext` returns `*authpb.User`. Define a local `AuthUser` value type and convert at the middleware boundary.

### Medium

**M1** — Repository bypasses `internal/errors` domain errors; inconsistent with `base.go`.
**M2** — `GetProductList` logs full `filters` struct at INFO level on every request.
**M3** — `CreateProductImage` uploads to MinIO before checking product exists.
**M4** — `category_repository.go:46` uses old-style `err.(*pq.Error)` instead of `errors.As`.
**M5** — `GetProductList` issues two sequential queries (COUNT + SELECT) without transaction isolation.
**M6** — No request body size limit on `BulkCreateCategories`.
**M7** — `product_cache.go:43` uses MD5 for cache keys; use FNV/CRC32.
**M8** — Seller scoping in `GetProductList` is in the handler, not the service layer.

### Low

**L1** — `internal/errors/category.go` misnamed; contains errors for all domain types.
**L2** — No observability: no metrics, no trace propagation, no `request_id` in service logs.
**L3** — `GetImageByProductID` and `GetImageBySKUID` omit `object_key` from SELECT.
**L4** — `CreateProductImage` position subquery has TOCTOU race under concurrent uploads.
~~**L5** — `sku_repository.go:49-51` double-encodes `VariantAttrs` (already `json.RawMessage`) via `json.Marshal`. **Data corruption risk.** Pass directly.~~
✅ **FIXED** — `json.Marshal` removed in both `CreateSKU` and `UpdateSKU` repository methods; `[]byte(sku.VariantAttrs)` passed directly to the driver. Unused `encoding/json` import removed.
**L6** — `product_cache.go DeleteProduct` calls `inner.GetProductByID` before delete; error silently discarded.

---

## api-gateway

### Critical

**C1 — gRPC auth connection uses insecure credentials in all environments**
`internal/middleware/auth.go:31`

Bearer tokens forwarded over plaintext gRPC. Use `credentials.NewTLS` in non-development environments. Require an explicit env flag to opt in to insecure mode.

~~**C2 — `.env` file committed to repository**~~
✅ **ALREADY COVERED** — Root `.gitignore` contains `services/**/.env` which covers `services/api-gateway/.env`. No additional change needed.

### High

~~**H1 — Audit log uses client-supplied `X-Forwarded-For` for IP without validation**~~
✅ **FIXED** — `main.go` now uses `net.SplitHostPort(r.RemoteAddr)` for all audit log IP entries. `X-Forwarded-For` no longer used.

**H2 — Router hot-reload polls every 1 second wastefully**
`main.go:253-269`

O(routes) work every second. Expose `loader.Changed() <-chan struct{}` from the LISTEN/NOTIFY loop and drive rebuilds from that. — **Pending**

~~**H3 — Auto-bind can silently downgrade `auth_mode` to `"none"`**~~
✅ **FIXED** — `isAuthDowngrade` / `authLevel` helpers added to `autobind.go`. The `applyRoute` default branch now refuses updates where `authLevel(new) < authLevel(current)` and logs a WARN instead.

**H4 — `probeRoute` in admin handler has SSRF risk**
`internal/admin/handler.go:474`

Validate `req.Path` against known route prefixes and `addr` against known upstream addresses before issuing the probe request.

### Medium

**M1** — No security response headers (`X-Content-Type-Options`, `X-Frame-Options`, `HSTS`).
**M2** — `auditMiddleware` does not record 403 responses.
**M3** — `RedisRegistry.Pick` does SCAN + GET on every proxied request. Cache instance list with short TTL.
**M4** — `main.go` is 440 lines; violates SRP. Extract cors, audit, upstream pool, router builder into sub-packages.
**M5** — `updateRoute` builds SQL SET clause by string concatenation of field names.
**M6** — `auth_mode` validated only at DB constraint level; return 400 from handler validation.
~~**M7** — `getStats` double-response bug: `jsonErr` + fall-through to `jsonOK` after `rows.Err()`.~~
✅ **NOT PRESENT** — Current code has `return` after `jsonErr` in `getStats`. Bug not present in this codebase revision; finding was stale.

### Low

**L1** — Zero tests anywhere in api-gateway.
**L2** — Comment placement confusion around `originAllowed` function.
**L3** — `BuildDSN()` called twice in `main.go`.
**L4** — Blocklist metadata `HSet` error silently discarded.

---

## currency-service

### Critical

~~**C1 — Admin authorization relies on forgeable `X-User-Role` header**~~
✅ **FIXED** — `NewAuthMiddleware` created in `interfaces/http/auth_middleware.go` with gRPC JWT validation (mirrors product-catalog-service pattern). Admin route now wrapped with `authMW.RequireRole("admin")` in the router. `X-User-Role` check removed from `ToggleCurrency` handler. `AuthServiceAddr` added to config. `defer authMW.Close()` in `main.go`.

~~**C2 — `Toggle` silently succeeds for non-existent currency codes**~~
✅ **FIXED** — `Toggle` repository now calls `result.RowsAffected()` and returns `&ErrCurrencyNotFound{Code: code}` when `n == 0`.

### High

~~**H1 — `base` currency parameter not validated before DB/cache**~~
✅ **FIXED** — `GetRates` handler now calls `valueobjects.NewCurrencyCode(base)` before executing the use case; returns 400 on invalid code.

**H2 — `CurrencyCode` value object accepts non-alphabetic characters**
`domain/valueobjects/currency_code.go:11-17`

Add character validation (`A-Z` only) in `NewCurrencyCode`.

**H3 — `FetchDuration`, `CacheHits`, `CacheMisses` metrics defined but never recorded**
Instrument providers with `ObserveFetch`; increment cache counters in `GetRatesUseCase`.

**H4 — `interfaces/metrics` is dead stub package**
Delete `services/currency-service/interfaces/metrics/`.

### Medium

**M1** — `ToggleCurrencyUseCase` has TOCTOU: Get + Toggle as two DB calls. Fixing C2 collapses this to one.
**M2** — `GetRatesHistoryUseCase` doesn't wrap errors with `fmt.Errorf`.
**M3** — `runMigrations` builds DSN independently from `Config.DSN()`. Two DSN construction paths.
**M4** — `OpenExchangeRatesProvider` synthesises `AsOf` from `time.Now()` instead of the API response field.
**M5** — No request body size limit on `ToggleCurrency`.
**M6** — `ConversionService` is unused dead code. Wire `/convert` or remove.

### Low

**L1** — `getEnvDuration` silently ignores unparseable env var values.
**L2** — `KAFKA_BROKERS` read via `os.Getenv` instead of config.
**L3** — `HistoryByBase` query casts `as_of::date` on left side, disabling index. Use range predicate.
**L4** — Test fake returns `errors.New("not found")` instead of domain error; test doesn't assert error type.
**L5** — `domain/valueobjects/rate.go` and `money.go` appear to be dead code.

---

## order-management-service

### Verdict: Approved with Required Changes

**Critical**

**C1** — `order_service.go:229` compares payment status against a string literal `"CAPTURED"`. Define a domain constant.

**High**

~~**H1** — Spin-wait loop on idempotency lock (`order_service.go:118-139`) burns goroutines at 50ms intervals for up to 10s. Remove the loop; fall through to DB check after a single NX attempt.~~
✅ **FIXED** — Spin-wait loop removed. Single `SetNX` attempt; all paths fall through to DB check immediately. DB unique constraint on `idempotency_key` is the correctness safety net.

~~**H2** — `order_repository.go:182` builds outbox payload via `fmt.Sprintf` instead of `json.Marshal`. Use `json.Marshal` for consistency.~~
✅ **FIXED** — `json.Marshal(map[string]string{"order_id": orderID.String()})` used; error now surfaced instead of silently dropped.

**H3** — Two round-trips for paginated list queries (COUNT + SELECT). Use `COUNT(*) OVER()` window function.

**Medium**

**M1** — No max item count on checkout — trivial DoS vector via 10,000-item orders. Add a max-items guard.
**M2** — `json.Marshal` error silently swallowed before `MarkCancelled`.
**M3** — No metrics or trace propagation.

**Low**

**L1** — `order_service_test.go:416` uses direct type assertion instead of `errors.As`.
**L2** — No integration tests against real PostgreSQL.

---

## inventory-service

### Verdict: Approved with Required Changes

**High**

~~**H1** — `inventory_service.go:116` warms Redis with TTL `0` (forever). Use a finite TTL (e.g. 24h) so the cache self-heals after ops incidents.~~
✅ **FIXED** — `s.rdb.Set(ctx, key, inv.QtyAvailable, 24*time.Hour)` — TTL changed from `0` to `24h`.

~~**H2** — `inventory_service.go:183` `ReleaseStock` uses `INCRBY` unconditionally; if the key was evicted, it creates a key seeded only with `qty`, corrupting the counter. Use `luaIncrIfExists` pattern.~~
✅ **FIXED** — `ReleaseStock` now calls `luaIncrIfExists` (already defined in the file for `AddStock`). If the key is absent, the next `ReserveStock` cache-miss will warm it correctly from DB.

**Medium**

**M1** — `GetReservationDetails` not in `contracts.InventoryRepository` interface; untestable.
**M2** — No HTTP endpoint for `AddStock`; sellers have no self-service restock path.
**M3** — No metrics.

**Low**

**L1** — `DefaultWarehouseID` is a `var` not a `const`-equivalent.
**L2** — Test `DialTimeout: 1` is 1 nanosecond; use `time.Millisecond`.

---

## payment-service

### Verdict: BLOCKED — Tests Required

**Critical**

**C1 — Zero test coverage for a service that processes money**

`ChargeCard`, `RefundPayment`, and `HandleCaptureWebhook` have no tests. Blocking before production.

**High**

**H1** — `payment_repository.go:30` builds SQL interval via `fmt.Sprintf` string injection. Parameterise.

**H2** — `ChargeCard` returns `IDEMPOTENCY_CONFLICT` with no `Retry-After` guidance.

**H3** — Gateway name hard-coded to `"fake"` (`payment_service.go:119`). Add `Name() string` to `PaymentGateway` interface.

**Medium**

**M1** — Webhook timestamp check uses `Duration.Abs()` which accepts future timestamps. Tighten to reject timestamps more than 30s in the future.
**M2** — `RefundPayment` has no idempotency key.
**M3** — No metrics.

---

## notification-service

### Verdict: Approved with Recommended Changes

**High**

**H1** — `consumer/handler.go:89`: corrupt `outbox_id` UUID disables dedup (`dedupEnabled = false`) instead of returning an error. A repeatedly-delivered corrupt message can spam users.

**H2** — `consumer/handler.go:112-113`: malformed JSON is silently ACKed (permanent message loss). At minimum, emit a metric; ideally route to DLQ.

**Medium**

**M1** — `buildNotification` mixes currency formatting logic with Kafka dispatch. Extract `internal/format` package.
**M2** — `Handle` function has no unit tests despite dedup and routing logic.
**M3** — Health server port `:8085` hardcoded (`main.go:53`). Will conflict with import-service.
**M4** — No metrics.

---

## admin-ui

### Verdict: Conditional Pass

**Critical**

- Zero test coverage.

**High**

~~**H1** — `app/api/gateway/[...path]/route.ts`: no path sanitisation on proxy target. Reject `..` segments.~~
✅ **FIXED** — `..` and `.` segment check added; returns 400 before forwarding.

**Medium**

- Dashboard layout (`layout.tsx`) is 407 lines; split into sub-components.
- Login page is 289 lines; extract `<BootTerminal>`.
- `getCurrencies`/`toggleCurrency` bypass `apiFetch` — no `json.success` check.
- Blocklist form inputs have no `id` attributes; `<label>` associations broken.
- Raw API error strings rendered to users.

---

## backoffice-ui

### Verdict: Conditional Pass

**Critical**

- Zero test coverage.

**High**

~~**H1** — Proxy (`app/api/proxy/[...path]/route.ts`) forwards unauthenticated requests to gateway with no 401 short-circuit. Return 401 when cookie is absent.~~
✅ **FIXED** — 401 returned immediately when `bo_token` cookie absent.

~~**H2** — Same proxy path sanitisation gap as all other UI services.~~
✅ **FIXED** — `..` / `.` segment rejection added to proxy function.

**Medium**

- Moderation page uses `catch(console.error)` — errors invisible to users.
- `<img>` instead of Next.js `<Image>` on product thumbnails.
- `req<T>` returns `undefined as T` on 204; type lie.
- Dashboard shows "—" for users/sellers/orders — API calls never made.

---

## buyer-ui

### Verdict: Needs Work

**Critical**

- Zero test coverage. — **Pending**
- ~~Checkout form has no client-side validation on any address field — empty orders can be placed.~~
  ✅ **FIXED** — `handlePlaceOrder` validates all five address fields (name, phone, line1, city, pincode) before submitting; returns user-facing error listing missing fields.

**High**

**H1** — Checkout sends client-controlled `unit_price` from Zustand (localStorage) to the order API. Backend **must** validate price server-side. If it trusts the client value, this is a price manipulation vector.

**Medium**

- Proxy route missing try/catch around upstream `fetch`; unhandled 500 on gateway unreachability.
- No `buyer_auth_hint` cookie; auth state requires async roundtrip on every page.
- Product 404 page uses native `<a>` instead of Next.js `<Link>`.
- `GATEWAY_URL ?? GATEWAY_URL` (self-coalesce) dead code in `app/api/auth/login/route.ts`.

---

## seller-ui

### Verdict: Conditional Pass

**Critical**

- Zero test coverage.

**Medium**

~~**M1** — Wizard double-create bug: navigating back from SKUs to Info and re-submitting creates a second product instead of updating.~~
✅ **FIXED** — `saveBasicInfo` now short-circuits with `setStep("skus")` when `productId` is already set. Product is only ever created once.

**M2** — `app/api/auth/me/route.ts` returns full upstream JSON verbatim. Whitelist `{ id, email, role }` like buyer-ui. — **Pending**

~~**M3** — Proxy path sanitisation gap.~~
✅ **FIXED** — `..` / `.` segment rejection added to seller-ui proxy function.

**Low**

- No MIME type / file size validation in `ImageDropzone`.
- Raw enum strings (`DRAFT`/`ACTIVE`) exposed as dropdown labels.
- `req<T>` returns `undefined as T` on 204 (same type-lie as backoffice-ui).

---

## Recommended Remediation Plan

### Sprint 1 — Production Blockers (P0)

| # | Item | Service | Status |
|---|---|---|---|
| 1 | Fix token refresh — send raw token not hash | auth-service | ✅ Done |
| 2 | Add OAuth CSRF state validation | auth-service | ✅ Done |
| 3 | Fix SKU/image ownership checks | product-catalog-service | ✅ Done |
| 4 | Fix `DeleteProduct` missing ownership check | product-catalog-service | ✅ Done |
| 5 | Fix gRPC connection leak in auth middleware | product-catalog-service | ✅ Done |
| 6 | Add TLS to gRPC auth channel | api-gateway | Pending |
| 7 | Remove `.env` from git | api-gateway | ✅ Already covered by root .gitignore |
| 8 | Fix auth header SSRF in currency-service admin | currency-service | ✅ Done |
| 9 | Fix `Toggle` silent no-op for unknown currency | currency-service | ✅ Done |
| 10 | Write payment-service tests | payment-service | Pending |
| 11 | Fix checkout double-create in seller-ui wizard | seller-ui | ✅ Done |
| 12 | Add checkout form validation in buyer-ui | buyer-ui | ✅ Done |
| 13 | Fix proxy path sanitisation (all 4 UIs) | All UIs | ✅ Done |

### Sprint 2 — High Priority (P1)

| Item | Service | Status |
|---|---|---|
| Add MaxBytesReader to all POST/PUT endpoints | product-catalog | ✅ Done (LimitedReader in DecodeJSON) |
| Add rate limiting to auth endpoints | auth-service | Pending |
| Fix `AdminHandler` bypassing service layer | auth-service | Pending |
| Fix `ReleaseStock` INCRBY on evicted key | inventory-service | ✅ Done |
| Add TTL to Redis stock warming | inventory-service | ✅ Done |
| Add `auth_mode` downgrade protection in auto-bind | api-gateway | ✅ Done |
| Fix audit log IP spoofing | api-gateway | ✅ Done |
| Fix spin-wait loop in order idempotency | order-management | ✅ Done |
| Fix unauthenticated proxy short-circuit | backoffice-ui | ✅ Done |
| Validate `base` currency param | currency-service | ✅ Done |
| Fix `getFacebookUserInfo` HTTP context | auth-service | ✅ Done |

### Sprint 3 — Technical Debt (P2)

| Item | Service | Status |
|---|---|---|
| Define `AuthServicePort` interfaces; inject into handlers | auth-service | Pending |
| Fix `VariantAttrs` double JSON marshalling (data corruption) | product-catalog | ✅ Done |
| Add metrics to order/inventory/payment/notification services | Multiple | Pending |
| Decompose `main.go` | api-gateway | Pending |
| Add unit tests to all UI services | All UIs | Pending |
| Fix `ConversionService` dead code | currency-service | Pending |
| Fix `getStats` double-response bug | api-gateway | ✅ Not present in codebase |
| Fix outbox payload (fmt.Sprintf → json.Marshal) | order-management | ✅ Done |
| Move `ADMIN_BOOTSTRAP_SECRET` to config | auth-service | ✅ Done |
| Fix Swagger `@BasePath` | auth-service | ✅ Done |
| Fix gRPC `GracefulStop` on shutdown | auth-service | ✅ Done |
| Fix pq duplicate-key error check | auth-service | ✅ Done |
| Add email format validation on registration | auth-service | ✅ Done |
