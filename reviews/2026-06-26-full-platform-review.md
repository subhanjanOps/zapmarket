# ZapMarket Platform — Production Code Review

**Date:** 2026-06-25
**Scope:** All services — auth-service, product-catalog-service, order-management-service, inventory-service, payment-service, notification-service, currency-service, api-gateway, buyer-ui, seller-ui, admin-ui, backoffice-ui
**Reviewer:** Principal Engineer

---

## Executive Summary

ZapMarket's backend services demonstrate a generally sound architectural foundation: layered Clean Architecture, repository interfaces with domain contracts, structured logging, transactional outbox patterns, and idempotency enforcement. However, the platform is not ready for production in its current state. Three systemic problems dominate the review: (1) the Dependency Inversion Principle is violated in every service that touches Redis or a concrete service struct, creating pervasive testability gaps; (2) zero automated test coverage exists across nine of the twelve services, meaning critical security and data-integrity paths are entirely unverified; (3) all three UI services leak internal Docker network addresses into client-side JavaScript bundles — a consistent infrastructure topology disclosure. Three backend services carry data-integrity bugs severe enough to corrupt ledger state or produce zombie orders under failure conditions that are not merely theoretical. The platform requires focused remediation across two sprint cycles before a production release is responsible.

---

## Overall Verdict by Service

| Service | Critical | High | Medium | Low | Verdict |
|---|---|---|---|---|---|
| auth-service | 1 | 3 | 4 | 4 | CHANGES_REQUIRED |
| product-catalog-service | 1 | 3 | 5 | 4 | CHANGES_REQUIRED |
| order-management-service | 1 | 4 | 4 | 4 | CHANGES_REQUIRED |
| inventory-service | 0 | 2 | 4 | 3 | APPROVED_WITH_RECOMMENDATIONS |
| payment-service | 1 | 4 | 3 | 2 | CHANGES_REQUIRED |
| notification-service | 0 | 3 | 3 | 4 | CHANGES_REQUIRED |
| currency-service | 0 | 3 | 5 | 4 | CHANGES_REQUIRED |
| api-gateway | 0 | 3 | 5 | 4 | CHANGES_REQUIRED |
| buyer-ui | 1 | 4 | 5 | 4 | CHANGES_REQUIRED |
| seller-ui | 2 | 5 | 7 | 5 | CHANGES_REQUIRED |
| admin-ui | 1 | 4 | 5 | 4 | CHANGES_REQUIRED |
| backoffice-ui | 1 | 3 | 5 | 4 | CHANGES_REQUIRED |

**Platform Verdict: CHANGES_REQUIRED**

---

## Cross-Cutting Findings

The following anti-patterns appear in three or more services. Each represents a systemic engineering problem that should be resolved with a platform-wide decision, not service-by-service patches.

### XC-1: Concrete Infrastructure Types Injected into Service/Application Layers (DIP Violation)

**Affected services:** auth-service, order-management-service, inventory-service, notification-service, payment-service

Every service that uses Redis injects the concrete `*redis.Client` (or `*goredis.Client`) directly into the service struct rather than behind an interface. The same pattern appears for concrete service structs in auth-service handler layer.

- `order-management-service/internal/service/order_service.go:63` — `rdb *redis.Client`
- `inventory-service/internal/service/inventory_service.go:53` — `*goredis.Client`
- `notification-service/internal/consumer/handler.go:72` — `*redis.Client`
- `auth-service/internal/handler/http/handlers.go:92` — `*service.AuthService` (concrete)
- `auth-service/internal/handler/http/admin_handler.go:20` — `*service.AuthService` (concrete)

**Platform fix:** Define a minimal cache port interface per service (or share one from `pkg/cache`) with only the methods each service actually calls. The `redis.Client` satisfies every such interface automatically; tests inject an in-memory fake or `miniredis`. Establish this as a mandatory pattern in `docs/coding-guidelines.md`.

---

### XC-2: Zero or Near-Zero Test Coverage Across the Platform

**Affected services:** auth-service, product-catalog-service, order-management-service, payment-service, notification-service, api-gateway, buyer-ui, seller-ui, admin-ui, backoffice-ui

Ten of twelve services have either zero `*_test.go` / `*.test.ts` files or coverage limited to one or two trivial helpers. The `go:generate mockgen` directives in product-catalog-service and currency-service show intent, but no tests consume the generated mocks.

Critical untested paths include: JWT validation (auth-service), the checkout saga (order-management-service), payment capture + compensation (payment-service), rate-limit Lua script correctness (api-gateway), ownership enforcement (product-catalog-service), and Redis dedup gate (notification-service).

**Platform fix:** Establish a minimum 70% line coverage gate on `internal/service` and `internal/handler` packages enforced in CI. Add table-driven unit tests using mock interfaces before any service is declared production-ready. For UI services add Jest/Vitest + React Testing Library coverage for auth middleware, proxy allowlist, and data-fetch hooks.

---

### XC-3: Internal Gateway URL Leaked into Client JS Bundles via `NEXT_PUBLIC_`

**Affected services:** buyer-ui (`services/buyer-ui/next.config.ts:6`), seller-ui (`next.config.ts:6`), admin-ui (`next.config.ts:6`), backoffice-ui (`next.config.ts:6`)

All four Next.js frontends set `NEXT_PUBLIC_GATEWAY_URL` in the `next.config.ts` `env` block. When `NEXT_PUBLIC_GATEWAY_URL` is set to an internal Docker hostname (e.g. `http://zapmarket-api-gateway:8000`), Next.js bakes that value into the client JS bundle at build time. The BFF proxy pattern used by all four UIs means no client code needs the gateway URL — all browser requests go to `/api/proxy/*` or `/api/gateway/*` routes which read `GATEWAY_URL` server-side.

**Platform fix:** Remove the `env` block from all four `next.config.ts` files. Rename `NEXT_PUBLIC_GATEWAY_URL` to `GATEWAY_URL` (no prefix) in all route handlers and Docker Compose environment definitions. Document in `docs/engineering-standards.md` that internal service URLs must never carry the `NEXT_PUBLIC_` prefix.

---

### XC-4: Catch-All Proxy Routes with No Upstream Path Allowlist (SSRF / Privilege Escalation)

**Affected services:** seller-ui (`app/api/proxy/[...path]/route.ts:4`), admin-ui (`app/api/gateway/[...path]/route.ts:20`), backoffice-ui (`app/api/proxy/[...path]/route.ts:16`)

All three BFF services implement a wildcard catch-all proxy that forwards any client-supplied path to the internal gateway, attaching the user's JWT. The only guard present is a path-traversal check for `.` and `..` segments. An authenticated user can call any gateway endpoint — including admin routes, other sellers' resources, or internal debug endpoints — simply by crafting a URL.

**Platform fix:** Define an explicit `ALLOWED_PREFIXES` set per UI service containing only the upstream paths that UI is designed to access. Return HTTP 403 before any upstream fetch for paths outside the set. Treat this as a mandatory BFF pattern in `docs/architecture-principles.md`.

---

### XC-5: gRPC Inter-Service Channels Use No TLS

**Affected services:** product-catalog-service (`internal/middleware/auth.go:25`), api-gateway (`internal/middleware/auth.go:31`)

Both services call auth-service via gRPC using `insecure.NewCredentials()`. JWT bearer tokens are transmitted in cleartext over these channels. In any multi-node Kubernetes deployment without a service mesh enforcing mTLS, this exposes tokens to network interception.

**Platform fix:** Require TLS for all gRPC inter-service channels in non-development environments. Add `AUTH_SERVICE_TLS_CERT` to the config schema. Document the explicit network boundary assumption (loopback / same-pod) that makes cleartext acceptable in local development, behind an `APP_ENV=development` guard.

---

### XC-6: Hardcoded Default Credentials in Source Code and Config

**Affected services:** currency-service (`pkg/config/config.go:43`), api-gateway (`internal/routes/loader.go:169`), order-management-service (`.env:5`), auth-service, product-catalog-service (`.env.example` patterns)

`DB_PASSWORD` defaults to `"zappass123"` in at least two services' config loaders. Several `.env` files containing these credentials are tracked in git. Any misconfigured deployment silently authenticates with the shared development credential.

**Platform fix:** Remove all default values for `DB_PASSWORD`, `JWT_SECRET_KEY`, and similar secrets from config loaders. Validate their presence explicitly at startup and fail fast with a descriptive error. Add `*.env` (not just `.env.example`) to the root `.gitignore` and remove any currently tracked `.env` files with `git rm --cached`.

---

### XC-7: No Metrics Instrumentation in Backend Services

**Affected services:** inventory-service, notification-service, order-management-service, payment-service

The engineering standards require structured logs + metrics + trace IDs. None of the backend services (other than currency-service, which registers but never records three of its metrics) expose Prometheus counters or histograms. Critical signals — reservation failure rate, payment decline rate, notification dedup hits, cache miss rate — are invisible to alerting and dashboards.

**Platform fix:** Establish a shared `pkg/metrics` registry pattern (the currency-service `infrastructure/metrics/` package is the reference implementation). Mandate a minimum set of instrumented counters per service type: `requests_total{status}`, `errors_total{kind}`, and a domain-specific operation counter. Wire `/metrics` endpoint to the existing HTTP mux in each service.

---

### XC-8: No Distributed Trace ID Propagation

**Affected services:** order-management-service, notification-service, payment-service, api-gateway

Outgoing gRPC calls and Kafka messages do not carry a trace/correlation ID from the originating HTTP request context. Diagnosing a failed checkout across order-management, inventory, and payment service logs is currently impossible without manually correlating timestamps.

**Platform fix:** Extract the request ID from `middleware.GetReqID(ctx)` (chi) or generate a UUID at ingress, inject it into outgoing gRPC metadata as a standard key (`x-request-id`), and include it as a Kafka message header. Log it as a structured field (`"trace_id"`) in every operation log line.

---

## Critical Findings (All Services)

### auth-service — C-1

**Title:** AdminHandler directly calls UserRepository — bypasses service layer entirely
**File:** `internal/handler/http/admin_handler.go:25`

`AdminHandler` receives a `contracts.UserRepository` directly and calls `GetUserByID` + `UpdateUser` for the `UpdateUserRole` endpoint (lines 170–179). Business logic (fetch → mutate role → persist) is embedded in the HTTP handler. The interface layer must never call the repository tier directly.

**Fix:** Introduce an `AdminService` with methods `PromoteUser(ctx, id, newRole)` and `DeactivateUser(ctx, id)`. `AdminHandler` depends only on that service interface. Remove the `userRepo` field from `AdminHandler` entirely.

---

### product-catalog-service — C-1

**Title:** CreateSKU has no ownership check — any seller can add SKUs to another seller's product
**File:** `internal/handler/http/sku_handler.go:63`

`UpdateSKU` and `DeleteSKU` both call `assertOwnership` before mutating, but `CreateSKU` does not. A seller who knows any product UUID can POST `/v1/skus` with that `product_id` and successfully inject SKUs into a competitor's listing.

**Fix:** Call `assertOwnership(r.Context(), h.productService, req.ProductID, user)` in `CreateSKU` after decoding the request body, mirroring the pattern in `UpdateSKU` (line 219).

---

### order-management-service — C-1

**Title:** Silently-ignored `json.Marshal` errors can produce NULL outbox payload, violating NOT NULL constraint
**File:** `internal/service/order_service.go:197`

Lines 197, 208, and 228 use `cancelPayload, _ := json.Marshal(...)`. If Marshal fails, `cancelPayload` is nil, the `outbox.payload JSONB NOT NULL` constraint fires, the transaction rolls back, and the order stays RESERVED while inventory has already been released — a zombie state with no outbox event emitted.

**Fix:** Check every Marshal error and return early: `cancelPayload, marshalErr := json.Marshal(...); if marshalErr != nil { return nil, pkgerrors.NewInternal(...) }`. Apply to lines 197, 208, and 228.

---

### payment-service — C-1

**Title:** MarkFailed has no status guard — CAPTURED payments can be regressed to FAILED
**File:** `internal/repository/payment_repository.go:117`

The UPDATE in `MarkFailed` uses `WHERE id = $1 AND deleted_at IS NULL` with no status restriction. A late failure webhook can transition an already-CAPTURED or REFUNDED payment to FAILED, corrupting the double-entry ledger and firing a spurious `payment.failed` outbox event.

**Fix:** Add `AND status IN ('PENDING', 'AUTHORISED')` to the UPDATE WHERE clause, matching the guard in `MarkCaptured`. Return `pkgerrors.NewConflict` when 0 rows are affected.

---

### buyer-ui — C-1

**Title:** Internal gateway URL baked into client bundle via `NEXT_PUBLIC_` env override
**File:** `services/buyer-ui/next.config.ts:6`

The `env` block propagates `NEXT_PUBLIC_GATEWAY_URL` into the client JS bundle. If an operator sets this to the internal Docker address, every browser receives the internal network topology. Server components already read `process.env.GATEWAY_URL` directly; clients should never need the gateway URL.

**Fix:** Remove the `env` block from `next.config.ts` entirely. See XC-3 for the platform-wide fix.

---

### seller-ui — C-1

**Title:** Unrestricted catch-all proxy — no upstream path allowlist (SSRF / privilege escalation)
**File:** `app/api/proxy/[...path]/route.ts:4`

The proxy forwards any authenticated request verbatim to `${GW}/${pathSegments.join('/')}` with the seller's JWT. A client-side bug, XSS payload, or curious developer can reach `/api/proxy/v1/admin/users` or any other gateway route. See XC-4 for the platform-wide pattern.

**Fix:** Define an explicit set of allowed upstream path prefixes in `lib/proxy-allowlist.ts` and return 403 for any path not on the list.

---

### seller-ui — C-2

**Title:** No seller-role check after login — buyers and admins can obtain a `seller_token` cookie
**File:** `app/api/auth/login/route.ts:28`

The login BFF sets `seller_token` whenever `upstream.ok` is true, without inspecting `data.role`. Any valid ZapMarket account — buyer, admin, unverified user — receives a `seller_token` cookie and gains access to the seller dashboard and product management endpoints.

**Fix:** After parsing the upstream response, check `data.role === 'seller'`. Return 403 with `{ error: 'This portal is for sellers only' }` and set no cookie if the role is wrong.

---

### admin-ui — C-1

**Title:** `getCurrencies` / `toggleCurrency` bypass `apiFetch` — data correctness bug
**File:** `lib/api.ts:206`

Both currency functions call `fetch` directly. `getCurrencies` calls `res.json()` raw, returning the gateway envelope object rather than the `Currency[]` array — `CurrenciesPage` maps over `undefined` at runtime. `toggleCurrency` returns void without checking the `success` field. They also use a different URL pattern (`/api/gateway/api/v1/...`) that may hit the wrong upstream path.

**Fix:** Migrate both to `apiFetch` with consistent paths: `export const getCurrencies = () => apiFetch<Currency[]>('/gateway/v1/currencies')`.

---

### backoffice-ui — C-1

**Title:** Proxy route has no upstream path allowlist — authenticated SSRF
**File:** `app/api/proxy/[...path]/route.ts:16`

The catch-all proxy blindly forwards any path to the internal gateway. Only `.` and `..` segments are blocked. An authenticated admin can reach any gateway endpoint. See XC-4 for the platform-wide pattern.

**Fix:** Introduce an `ALLOWED_PREFIXES` set and reject requests whose path does not match, returning 403.

---

## High Priority Findings

### auth-service

**H-1 — HTTP and gRPC handlers call `crypto.GenerateAccessToken` directly**
`internal/handler/http/handlers.go:257` (and lines 303, 419, 552, 636) and `internal/handler/grpc/auth_server.go:256`. The interface layer performs infrastructure work (JWT signing) that belongs in the service layer. Fix: Add `IssueAccessToken(ctx, user) (string, error)` to `AuthService`; handlers call only this method.

**H-2 — Handlers depend on concrete `*service.AuthService`, not an interface**
`internal/handler/http/handlers.go:92`, `admin_handler.go:20`, `oauth_service.go:34`. Prevents unit testing via mock injection. Fix: Define `AuthServiceInterface` in `domain/contracts` and wire the concrete struct at startup.

**H-3 — OAuth access tokens sent as URL query parameters**
`internal/service/oauth_service.go:193` (Google) and line 228 (Facebook). Tokens in URLs are recorded in server logs and browser history (CWE-598). Fix: Use `req.Header.Set("Authorization", "Bearer "+token.AccessToken)` and remove the token from the URL.

---

### product-catalog-service

**H-1 — `productListKey` silently swallows `json.Marshal` error, collapsing all list cache keys**
`internal/service/product_cache.go:43`. `b, _ := json.Marshal(filters)` — if Marshal fails, `b` is nil, `md5.Sum(nil)` produces a fixed hash, and all filter combinations share one cache key, silently returning stale data. Fix: Check the error and return a fallback key or skip the cache.

**H-2 — Update handlers return locally-constructed stale objects, not DB state**
`internal/handler/http/product_handler.go:325`, `category_handler.go:270`, `sku_handler.go:253`. Clients receive `updated_at` timestamps older than the actual row. For optimistic-lock products, the next update from the client will 409 immediately. Fix: Re-fetch via `GetProductByID` after successful update, or use `RETURNING updated_at`.

**H-3 — Zero test coverage across the entire service**
No `*_test.go` files anywhere. The `go:generate mockgen` directives exist but nothing consumes them. Fix: Add table-driven unit tests for service-layer validation, the `BulkCreateCategories` topological sort, and handler-layer tests via `httptest` targeting `CreateSKU` and `UpdateProduct`.

---

### order-management-service

**H-1 — gRPC `ClientConn` never stored or closed — resource leak on shutdown**
`internal/clients/inventory_client.go:18`. `conn` is discarded after passing to the pb service constructor. `main.go` has no reference to call `conn.Close()`. Fix: Store `conn` in the client struct, add `Close() error`, and defer both client closes in `main.go` after HTTP shutdown.

**H-2 — Checkout saga gap: `DeductStock` can succeed before `MarkConfirmed` fails — stock permanently lost**
`internal/service/order_service.go:216`. Units removed from `qty_on_hand` while the order remains RESERVED. A retry re-attempts reservation against already-deducted stock. Fix: Move `DeductStock` calls to after a successful `MarkConfirmed`, or remove the direct call entirely and trigger it via the `order.confirmed` Kafka event.

**H-3 — `Checkout` function is 163 lines, far exceeding the 80-line maximum**
`internal/service/order_service.go:81–244`. Seven distinct responsibilities in one method. Fix: Extract `checkIdempotency()`, `buildDomainItems()`, `reserveAllStock()`, `finaliseCheckout()` helpers; `Checkout` becomes a ~30-line orchestrator.

**H-4 — No maximum item count enforced — unbounded serial gRPC calls on checkout**
`internal/service/order_service.go:168`. 10,000 items = 10,000 serial round-trips to inventory-service. Fix: Add a `maxCheckoutItems` constant (e.g. 50) and validate in the handler before the saga begins.

---

### inventory-service

**H-1 — Concrete `*goredis.Client` injected into service layer**
`internal/service/inventory_service.go:53`. See XC-1. Fix: Define `StockCachePort` interface in `internal/domain/contracts/`; implement it in `internal/infrastructure/cache/redis_stock_cache.go`.

**H-2 — `GetReservationDetails` declared in interface but never called — dead interface method**
`internal/domain/contracts/repositories.go:53`. Widens the contract unnecessarily and forces all future mocks to stub it. Fix: Remove from the interface and concrete repository.

---

### payment-service

**H-1 — Gateway charge succeeds but `MarkCaptured` failure leaves an orphaned charge**
`internal/service/payment_service.go:127`. Customer's card is debited but the payment record stays PENDING. On retry, `GetByIdempotencyKey` finds PENDING and returns it, stalling the order saga permanently. Fix: On `MarkCaptured` failure, attempt `gateway.Refund` as compensation and call `MarkFailed`.

**H-2 — `GatewayResponse` and `DeletedAt` never populated — silent data loss**
`internal/repository/payment_repository.go:191`. `paymentSelectQuery` selects 12 columns; `domain.Payment` has 14 fields. `Payment.GatewayResponse` is always nil regardless of DB state. Fix: Add `gateway_response, deleted_at` to the SELECT and the corresponding `Scan` targets.

**H-3 — `GetPendingWithGatewayCharge` not declared in `PaymentRepository` interface**
`internal/repository/payment_repository.go:25`. Unreachable dead code through the interface. Fix: Either add it to `contracts.PaymentRepository` with a service method, or delete it.

**H-4 — Zero test coverage**
No `*_test.go` files. The two bugs in C-1 and H-1 would have been caught by table-driven unit tests with mock implementations. Fix: Cover idempotency replay, successful charge, declined charge, `MarkCaptured` failure compensation, refund paths, and HMAC webhook validation.

---

### notification-service

**H-1 — TOCTOU race in dedup gate causes duplicate notifications**
`internal/consumer/handler.go:98`. Redis `Exists` check and `SetNX` after send are not atomic. Two concurrent consumers can both pass `Exists` before either sets the key. Fix: Use `SetNX` before sending. If `SetNX` returns false, return nil immediately. On send failure, `Del` the key so the next retry can reclaim it.

**H-2 — Handler depends on concrete `*redis.Client`**
`internal/consumer/handler.go:72`. See XC-1. Fix: Define a minimal `deduplicator` interface with `Exists`, `SetNX`, `Del` methods; inject it into `Handler`.

**H-3 — No metrics instrumentation**
`main.go:1`. No counters for `events_consumed_total`, `notifications_sent_total`, `dedup_hits_total`, `notification_errors_total`. Silent consumer failure is undetectable in production. Fix: Add a `pkg/metrics` adapter; expose `/metrics` alongside `/health`.

---

### currency-service

**H-1 — Interface layer imports concrete `*metrics.Metrics` (layer violation)**
`interfaces/http/router.go:9`. Clean Architecture forbids the interfaces layer from depending on the infrastructure layer. Fix: Define a `MetricsRecorder` interface in `interfaces/http` and inject it instead of the concrete type.

**H-2 — Interface layer imports concrete `*metrics.Metrics` in worker**
`interfaces/worker/rate_ingestor.go:9`. Same DIP violation as H-1. Fix: Define an `IngestMetricsRecorder` interface in the worker package with `RecordIngest(success bool)`.

**H-3 — Auth error responses use `text/plain` Content-Type despite JSON body**
`interfaces/http/auth_middleware.go:52`. `http.Error()` sets `text/plain`; clients inspecting Content-Type will not attempt JSON decoding. Fix: Replace `http.Error` calls with a helper that sets `Content-Type: application/json` before writing, consistent with `Handler.writeError`.

---

### api-gateway

**H-1 — gRPC auth-service connection uses no TLS**
`internal/middleware/auth.go:31`. JWTs transmitted in plaintext. See XC-5. Fix: Use mTLS or `credentials.NewTLS(&tls.Config{})` with a CA cert; gate cleartext on `APP_ENV=development`.

**H-2 — `main()` is ~298 lines — massive SRP violation**
`main.go:35`. Inline infrastructure wiring, router, CORS, audit middleware, signal handling, and graceful shutdown. Fix: Extract `newServer(cfg, deps)`, `newRouter(deps)`, `runWithGracefulShutdown(ctx, srv, deps)`; move middleware factories to the middleware package.

**H-3 — Client-supplied `X-Request-ID` trusted without validation**
`internal/middleware/requestid.go:18`. Arbitrary values (SQL fragments, oversized strings) are accepted verbatim and stored in audit logs (audit log poisoning). Fix: Accept the header only if `uuid.Parse(id) == nil`; otherwise generate a fresh UUID.

---

### buyer-ui

**H-1 — `payment_method` field missing from order request body**
`services/buyer-ui/app/checkout/page.tsx:175`. `paymentMethod` state is captured but never included in `handlePlaceOrder` body. The UI selector is a no-op. Fix: Add `payment_method: paymentMethod` to the request body object.

**H-2 — Duplicate GW constant defined in three places**
`app/page.tsx:33`, `lib/api.ts`, `lib/gateway.ts`. Any change to the default fallback URL must be made in three places. Fix: Make `lib/gateway.ts` the sole source; all files import `{ GW }` from it.

**H-3 — Local `publicImageUrl` in deals page duplicates `lib/images.ts`**
`services/buyer-ui/app/deals/[slug]/page.tsx:10`. Local version has different logic that breaks for MinIO-hosted images. Fix: Remove the local function and import from `@/lib/images`.

**H-4 — `--font-syne` CSS variable referenced but Syne font never loaded**
`services/buyer-ui/app/deals/[slug]/page.tsx:81` and `app/account/orders/[id]/page.tsx:222`. Root layout only loads Roboto. Browser silently falls back to system font. Fix: Add Syne to root layout via `next/font/google`, or replace references with the correct loaded font variable.

---

### seller-ui

**H-1 — `seller_auth_hint` cookie not cleared on logout**
`app/api/auth/logout/route.ts:5`. Only `seller_token` is deleted. Client-side code reading `seller_auth_hint` reports the user as authenticated after logout. Fix: Add `res.cookies.delete('seller_auth_hint')` immediately after `seller_token` delete.

**H-2 — `NEXT_PUBLIC_GATEWAY_URL` propagated to browser bundle**
`next.config.ts:6`. See XC-3. Fix: Remove the `env` block; use `GATEWAY_URL` (server-only) in route handlers.

**H-3 — `fetchRates()` returns `stale: false` on total failure, masking broken currency conversion**
`lib/currency.tsx:82`. When the rates API fails and there is no localStorage cache, the catch block returns `{ rates: { USD: 1 }, asOf: '', stale: false }`. Revenue figures in non-USD currencies are silently wrong. Fix: Return `stale: true` from the total-failure branch.

**H-4 — `ImageDropzone` uses array index as React key — causes broken state on item removal**
`app/components/ImageDropzone.tsx:77`. `key={i}` causes React to misidentify DOM nodes when items are removed, restarting progress overlays. Fix: Add `id: crypto.randomUUID()` to `PendingFile`; use `key={p.id}`.

**H-5 — `handleFiles` captures stale `pending.length` in closure**
`app/components/ImageDropzone.tsx:29`. Concurrent uploads overwrite each other's progress state. Fix: After H-4 stable-key fix, replace index-based state updates with functional updates searching by `id`.

---

### admin-ui

**H-1 — `NEXT_PUBLIC_GATEWAY_URL` leaks internal infrastructure address to browser bundle**
`next.config.ts:6`. See XC-3.

**H-2 — `console.error` in `ErrorBoundary` leaks render stack traces in production**
`app/components/ErrorBoundary.tsx:15`. `componentDidCatch` dumps component stack to browser console. Fix: Route to a structured server-side log sink or guard behind `process.env.NODE_ENV !== 'production'`.

**H-3 — Proxy route has no path allowlist**
`app/api/gateway/[...path]/route.ts:20`. See XC-4.

**H-4 — No route-level `error.tsx` files — App Router error recovery broken**
`app/dashboard/`. A single render failure in any dashboard page takes down the entire layout. Fix: Add `app/dashboard/error.tsx` as a Client Component with `reset()` support per Next.js App Router convention.

---

### backoffice-ui

**H-1 — All API errors silently swallowed via `.catch(console.error)`**
`app/dashboard/products/page.tsx:72` and analogous lines in orders, moderation, users, and order detail pages. Empty tables are indistinguishable from API failures. Fix: Add `error` state per page; render an error banner in the catch block; replace `console.error` with a structured logger.

**H-2 — `NEXT_PUBLIC_GATEWAY_URL` leaks internal gateway address into client bundle**
`next.config.ts:6`. No client-side code reads this variable — all requests go through `/api/proxy/`. See XC-3.

**H-3 — Google Fonts preconnect present but font loaded via CSS `@import` — no preload benefit**
`app/layout.tsx:15`. The `@import` in `globals.css` creates a render-blocking chain. Preconnect hints are wasted. Fix: Move the `<link rel="stylesheet">` into `<head>` in `layout.tsx` and remove the `@import`, or migrate to `next/font/google`.

---

## Medium Priority Findings

### auth-service

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/service/auth_service.go:234` | OAuth CSRF protection silently disabled when Redis is unavailable — `ValidateAndConsumeOAuthState` returns `(true, nil)` when `rdb == nil`, removing all state validation |
| M-2 | `internal/handler/http/handlers.go:238` | No minimum password length on registration (72-char max enforced, but no minimum — inconsistent with `ResetPassword` which enforces 8 chars) |
| M-3 | `internal/service/oauth_service.go:197` | OAuth provider HTTP response status not checked before body parsing — 4xx/5xx errors produce misleading parse failures |
| M-4 | `cmd/main.go:129` | No rate limiting on `/login`, `/register`, `/password/reset` — exposes to credential-stuffing and enumeration attacks |

### product-catalog-service

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/repository/category_repository.go:46` | Uses old-style `err.(*pq.Error)` type assertion instead of `errors.As` — inconsistent with the rest of the repository layer |
| M-2 | `internal/handler/http/product_handler.go:91` | `ProductStatus` not validated at service/handler layer — invalid values hit the DB CHECK constraint, bypassing any future non-HTTP callers |
| M-3 | `internal/handler/http/sku_handler.go:70` | Nil `VariantAttrs` marshaled to JSON `"null"` instead of `"{}"` — inconsistent with product attributes default |
| M-4 | `internal/middleware/auth.go:25` | gRPC connection to auth-service uses insecure credentials — see XC-5 |
| M-5 | `internal/service/category_service.go:14` | `uniqueStrings` mutates input slice in place via `out := ss[:0]` — latent bug for any caller passing a reusable slice |

### order-management-service

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/service/order_service.go:63` | `orderService` takes concrete `*redis.Client` — see XC-1 |
| M-2 | `internal/service/order_service.go:81` | No trace ID propagation through saga steps — see XC-8 |
| M-3 | `internal/repository/order_repository.go:160` | `MarkReserved`/`MarkCancelled` do not assert current status before UPDATE — concurrent double-submit can resurrect a cancelled order |
| M-4 | `internal/repository/order_repository.go:59` | `rows.Err()` returned without context wrapping in `GetByUserID`, `GetOrderItems`, `GetBySellerID`, `ListAll` |

### inventory-service

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/service/inventory_service.go:89` | `ReserveStock` is 67 lines, exceeds 40-line limit; four distinct concerns interleaved |
| M-2 | `internal/service/inventory_service.go:1` | No metrics instrumentation — cache hits/misses and reservation outcomes invisible to alerting |
| M-3 | `go.mod:6` | `miniredis` listed as top-level production require — should appear only as a test dependency |
| M-4 | `internal/service/inventory_service.go:117` | Stock cache TTL hardcoded to 24 hours as magic constant in service layer |

### payment-service

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/service/payment_service.go:52` | `ChargeCard` is 104 lines — exceeds 80-line hard limit |
| M-2 | `internal/domain/models.go:42` | `LedgerDebit`/`LedgerCredit` use lowercase — inconsistent with project UPPER_SNAKE_CASE convention |
| M-3 | `internal/service/payment_service.go:119` | Gateway name `"fake"` hardcoded as string literal in service layer — should come from `gateway.Name()` |

### notification-service

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/consumer/handler.go:128` | Empty `UserID` not validated before dispatching notification — malformed events dispatch to empty address |
| M-2 | `main.go:107` | Flat 5-second Kafka retry with no backoff or cap — noisy log spam under sustained outage |
| M-3 | `internal/consumer/handler.go:19` | Currency formatting logic (`formatAmount`, `zeroDecimalCurrencies`) belongs in a domain/formatting package, not the consumer |

### currency-service

| ID | File | Issue |
|---|---|---|
| M-1 | `infrastructure/metrics/metrics.go:40` | `CacheHits`, `CacheMisses`, `FetchDuration` registered but never recorded — always read zero |
| M-2 | `application/usecases/get_rates.go:44` | Cache errors silently swallowed without logging — Redis outage invisible in logs |
| M-3 | `pkg/config/config.go:43` | Default DB password `"zappass123"` hardcoded as env-var fallback — see XC-6 |
| M-4 | `interfaces/http/router.go:31` | Health endpoint leaks internal connection error details in response body |
| M-5 | `interfaces/metrics/metrics.go:1` | Deprecated `interfaces/metrics` package not removed — empty file with deprecation comment |

### api-gateway

| ID | File | Issue |
|---|---|---|
| M-1 | `internal/routes/loader.go:169` | Default DB password hardcoded — see XC-6 |
| M-2 | `internal/admin/handler.go:76` | Admin handler queries `*sql.DB` directly with no service/repository layer |
| M-3 | `internal/middleware/blocklist.go:29` | Blocklist fails open on Redis error — security bypass under outage |
| M-4 | `internal/middleware/ratelimit.go:48` | Rate limiter undercounts concurrent requests within the same millisecond (ZADD member collision) |
| M-5 | `internal/` | Zero test coverage across entire service |

### buyer-ui

| ID | File | Issue |
|---|---|---|
| M-1 | `app/checkout/page.tsx:221` | Framer Motion `ease` arrays missing `as const` — TypeScript widens to `number[]` |
| M-2 | `app/checkout/page.tsx:237` | `StepIndicator` hardcoded to step 1 — never advances; purely decorative and misleading |
| M-3 | `app/account/orders/[id]/page.tsx:186` | Order item list uses array index as React key |
| M-4 | `app/error.tsx:15` | `console.error` in global error boundary instead of structured logging |
| M-5 | `middleware.ts:3` | `/api/proxy/v1/orders` and `/api/proxy/v1/wishlist` not in `PROTECTED` — no defense-in-depth for unauthenticated proxy calls |

### seller-ui

| ID | File | Issue |
|---|---|---|
| M-1 | `app/dashboard/layout.tsx:40` | `DashboardLayout` is a 474-line god component handling auth, theme, clock, currency, sidebar, drawer, scroll-lock |
| M-2 | `app/dashboard/layout.tsx:60` | Auth guard is client-side only — expired tokens pass middleware, causing a visible Splash flash before redirect |
| M-3 | `app/layout.tsx:11` | No React error boundaries — any unhandled throw crashes the entire app tree |
| M-4 | `app/layout.tsx:15` | Google Fonts preconnect present but no stylesheet link — Rubik, Outfit, DM Mono never load |
| M-5 | `app/dashboard/products/[id]/page.tsx:41` | `EditProductPage` initial fetch has no cancellation — `setState` called on unmounted component |
| M-6 | `app/dashboard/products/[id]/page.tsx:191` | Native `<img>` with `eslint-disable` instead of `next/image` with `unoptimized: true` |
| M-7 | `app/dashboard/products/new/page.tsx:12` | `slugify` function duplicated across two page files |

### admin-ui

| ID | File | Issue |
|---|---|---|
| M-1 | `app/login/page.tsx:30` | Theme FOUC — `data-theme` applied in `useEffect`, causing flash of unstyled content on every navigation |
| M-2 | `app/api/gateway/[...path]/route.ts:23` | Proxy unconditionally sets `Content-Type: application/json`, breaking non-JSON requests |
| M-3 | `app/dashboard/page.tsx:131` | `handleRefresh` uses `setTimeout` unrelated to actual fetch completion |
| M-4 | `app/dashboard/blocklist/page.tsx:148` | Destructive action buttons are icon-only with `title` but no `aria-label` |
| M-5 | `app/dashboard/layout.tsx:196` | Theme picker buttons suppress focus ring via `outline: none` — keyboard-inaccessible |

### backoffice-ui

| ID | File | Issue |
|---|---|---|
| M-1 | `app/dashboard/users/page.tsx:82` | Search fires a new API request on every keystroke — no debounce (products page correctly debounces 300ms) |
| M-2 | `app/dashboard/layout.tsx:214` | Client-side auth guard in `useEffect` causes hydration flash; middleware is the real enforcement point |
| M-3 | `app/components/Dialog.tsx:9` | Module-level mutable singletons for dialog state — unsafe for concurrent rendering; second mount overwrites first |
| M-4 | `app/dashboard/layout.tsx:216` | Theme flash on initial load — `localStorage` read in `useEffect` runs after first paint |
| M-5 | `app/dashboard/` | No `error.tsx` boundaries at any route segment — unhandled rejections crash the page |

---

## Low Priority Findings

### auth-service

| ID | File | Issue |
|---|---|---|
| L-1 | `internal/domain/models.go:105` | `domain.Claims` is a JWT infrastructure concept in the wrong layer |
| L-2 | `internal/domain/models.go:27` | `User.Role` stored as `string` instead of `domain.Role` type — allows arbitrary values at compile time |
| L-3 | `internal/handler/http/preferences_handler_test.go:1` | Only one test file for the entire service |
| L-4 | `internal/handler/grpc/auth_server.go:40` | Per-method gRPC logging boilerplate — should use a unary server interceptor |

### product-catalog-service

| ID | File | Issue |
|---|---|---|
| L-1 | `internal/handler/grpc/product_catalog_grpc_handler.go:113` | `time.Time.String()` used for proto timestamps instead of RFC3339Nano |
| L-2 | `internal/service/product_service.go:107` | Free-text `Search` field logged at Info level — PII-adjacent data in logs |
| L-3 | `internal/service/sku_service.go:72` | `GetSKUByID` missing nil UUID guard present in all other service methods |
| L-4 | `main.go:86` | Object storage client initialized with `context.Background()` and no timeout |

### order-management-service

| ID | File | Issue |
|---|---|---|
| L-1 | `.env:5` | `.env` file with credentials present on disk — should be gitignored |
| L-2 | `internal/domain/models.go:15` | `OrderPaid` status defined in FSM and DB enum but never set by the service |
| L-3 | `internal/handler/http/base.go:35` | No HTTP request body size limit on `DecodeJSON` |
| L-4 | `main.go:65` | `MigrateOnBoot` flag can cause race conditions in multi-replica deployments |

### inventory-service

| ID | File | Issue |
|---|---|---|
| L-1 | `internal/handler/grpc/inventory_grpc_handler.go:51` | Dead code — `reservation == nil` guard is unreachable |
| L-2 | `inventory-service.exe` | Windows binary committed to the repository |
| L-3 | `internal/domain/models.go:79` | No sweep job for expired reservations — `ExpiresAt` recorded but never acted on |

### payment-service

| ID | File | Issue |
|---|---|---|
| L-1 | `internal/handler/http/webhook_handler.go:106` | Webhook handler logs errors without a request/trace ID |
| L-2 | `internal/handler/http/webhook_handler.go:43` | `webhookPayload.Status` is an untyped string — compared against bare string literals |

### notification-service

| ID | File | Issue |
|---|---|---|
| L-1 | `internal/consumer/handler_test.go:1` | Core handler logic has zero test coverage — only `formatAmount` is tested |
| L-2 | `main.go:53` | Health server port hardcoded to `:8085` instead of read from config |
| L-3 | `go.mod:3` | `go 1.25.0` declared — version does not exist; causes toolchain resolution failures |
| L-4 | `internal/consumer/handler.go:79` | No trace ID propagation from Kafka message headers into context |

### currency-service

| ID | File | Issue |
|---|---|---|
| L-1 | `interfaces/grpc/server.go:44` | gRPC `GetRates` does not validate base currency code format (HTTP handler does) |
| L-2 | `application/usecases/get_rates_history.go:14` | `HistoryRepository` defined locally instead of reusing `domain/repositories.RatesRepository` |
| L-3 | `cmd/main.go:181` | `runMigrations` duplicates DSN construction instead of using a `MigrationDSN()` method |
| L-4 | `cmd/main.go:123` | Base currency `"USD"` hardcoded in worker startup — should be `RATE_BASE_CURRENCY` env var |

### api-gateway

| ID | File | Issue |
|---|---|---|
| L-1 | `internal/proxy/proxy.go:50` | No request body size limit on proxied requests |
| L-2 | `main.go:277` | Auto-binder activates implicitly on `APP_ENV=development` rather than requiring `GATEWAY_AUTO_BIND=true` |
| L-3 | `main.go:112` | `routes.BuildDSN()` called twice — should be captured in a variable |
| L-4 | `main.go:36` | `godotenv.Load()` error silently discarded — malformed `.env` indistinguishable from absent file |

### buyer-ui

| ID | File | Issue |
|---|---|---|
| L-1 | `app/checkout/page.tsx:441` | Promo code input missing `aria-label` |
| L-2 | `components/Navbar.tsx:235` | Mobile Sheet missing accessible dialog title |
| L-3 | `components/Navbar.tsx:55` | `activeCategory` nav state not derived from URL — desyncs on browser back/forward |
| L-4 | `lib/api.ts:5` | `apiFetch` returns `null` for both network errors and HTTP errors — hides status context from callers |

### seller-ui

| ID | File | Issue |
|---|---|---|
| L-1 | `app/dashboard/orders/page.tsx:39` | `fmtMoney` is a redundant pass-through alias for `formatFrom` in two files |
| L-2 | `app/dashboard/orders/page.tsx:51` | Icon-only buttons lack `aria-label` in orders page, `ImageDropzone`, and `SKUEditor` |
| L-3 | `app/components/CategoryPicker.tsx:353` | Prefill fetches up to 300 categories client-side to resolve a single ID |
| L-4 | `app/dashboard/loading.tsx:1` | `loading.tsx` unreachable because `DashboardLayout` is a Client Component |
| L-5 | `app/login/page.tsx:94` | Hardcoded marketing statistics will go stale |

### admin-ui

| ID | File | Issue |
|---|---|---|
| L-1 | `package.json:16` | `lucide-react@^1.20.0` is a non-existent version range |
| L-2 | `app/login/page.tsx:113` | Array index used as React key for terminal boot lines |
| L-3 | `package.json:1` | No `eslint-plugin-jsx-a11y` configured — accessibility issues not caught by lint |
| L-4 | `app/layout.tsx:1` | JetBrains Mono and Space Grotesk loaded without preconnect/preload via `next/font` |

### backoffice-ui

| ID | File | Issue |
|---|---|---|
| L-1 | `app/components/CommandPalette.tsx:38` | Shortcut hints `N→C`, `N→P` displayed but never handled |
| L-2 | `app/components/BulkIO.tsx:37` | CSV header parser splits on literal comma — breaks quoted headers |
| L-3 | `app/login/page.tsx:7` | Hardcoded stale statistics in login decorative panel |
| L-4 | `app/components/Dialog.tsx:81` | Enter key behavior in confirm dialog misleadingly commented |

---

## Recommended Remediation Plan

### Sprint 1 — Critical (Fix before next release)

These items represent data corruption risk, active security vulnerabilities, or authentication bypasses. No release should proceed without all of them resolved.

1. **[seller-ui C-2]** Add seller-role check to login BFF route (`app/api/auth/login/route.ts`). Buyers and admins must not receive a `seller_token`.
2. **[product-catalog-service C-1]** Add `assertOwnership` call to `CreateSKU` handler (`internal/handler/http/sku_handler.go:63`).
3. **[auth-service C-1]** Introduce `AdminService` and remove direct repository access from `AdminHandler` (`internal/handler/http/admin_handler.go`).
4. **[payment-service C-1]** Add `AND status IN ('PENDING', 'AUTHORISED')` to `MarkFailed` UPDATE (`internal/repository/payment_repository.go:117`).
5. **[order-management-service C-1]** Check `json.Marshal` errors on all compensation payloads in the checkout saga (`internal/service/order_service.go:197,208,228`).
6. **[admin-ui C-1]** Migrate `getCurrencies` and `toggleCurrency` to use `apiFetch` with consistent paths (`lib/api.ts:206`).
7. **[seller-ui C-1, admin-ui H-3, backoffice-ui C-1]** Implement upstream path allowlist in all three catch-all BFF proxy routes. See XC-4.
8. **[XC-3 — all UIs]** Remove `NEXT_PUBLIC_GATEWAY_URL` from all four `next.config.ts` `env` blocks.
9. **[XC-6]** Remove hardcoded `DB_PASSWORD` defaults from currency-service `pkg/config/config.go:43` and api-gateway `internal/routes/loader.go:169`. Fail fast if the env var is unset.
10. **[order-management-service, inventory-service, api-gateway .env files]** Ensure `.env` is gitignored and not currently tracked (`git ls-files .env`); run `git rm --cached` where needed.

---

### Sprint 2 — High (Fix within 2 weeks)

Security hardening, resource leaks, and data integrity issues that degrade correctness under realistic failure conditions.

1. **[XC-1]** Define cache port interfaces and remove concrete Redis injection from order-management-service, inventory-service, and notification-service service layers.
2. **[XC-5]** Configure TLS (or document the explicit loopback assumption with a code comment) for the product-catalog → auth-service and api-gateway → auth-service gRPC channels.
3. **[payment-service H-1]** Add gateway compensation path: on `MarkCaptured` failure, call `gateway.Refund` and `MarkFailed`.
4. **[payment-service H-2]** Add `gateway_response, deleted_at` to `paymentSelectQuery` and the `Scan` call.
5. **[order-management-service H-1]** Store and close gRPC `ClientConn` in inventory and payment clients.
6. **[order-management-service H-2]** Move `DeductStock` calls to after `MarkConfirmed` or remove the synchronous call in favour of Kafka-driven deduction.
7. **[order-management-service H-4]** Add `maxCheckoutItems` constant and validate in the handler.
8. **[notification-service H-1]** Fix TOCTOU dedup race — claim with `SetNX` before sending, `Del` on failure.
9. **[auth-service H-3]** Move OAuth token from URL query parameter to `Authorization` header in `getGoogleUserInfo` and `getFacebookUserInfo`.
10. **[api-gateway H-3]** Validate `X-Request-ID` as UUID before accepting it in `requestid.go`.
11. **[auth-service M-1]** Return explicit error (not `true, nil`) from `ValidateAndConsumeOAuthState` when Redis is unavailable.
12. **[auth-service M-4]** Add per-IP rate limiting middleware to `/login`, `/register`, and `/password/reset`.
13. **[notification-service H-2]** Define `deduplicator` interface in consumer package; inject instead of `*redis.Client`.
14. **[buyer-ui H-1]** Add `payment_method: paymentMethod` to `handlePlaceOrder` request body.
15. **[seller-ui H-1]** Delete `seller_auth_hint` cookie on logout.
16. **[seller-ui H-3]** Return `stale: true` from total-failure branch in `fetchRates()`.
17. **[product-catalog-service H-1]** Handle `json.Marshal` error in `productListKey`.
18. **[product-catalog-service H-2]** Re-fetch entity after successful update instead of returning locally-constructed stale objects.
19. **[backoffice-ui H-1]** Add `error` state to all listing pages; surface error banner on catch instead of `console.error`.
20. **[XC-2]** Begin minimum test coverage initiative: add table-driven unit tests for auth-service `AuthService`, payment-service charge/refund paths, order-management-service checkout saga, and notification-service `Handle` and `buildNotification`.

---

### Sprint 3 — Medium (Fix within 1 month)

Code quality, observability, and maintainability items that reduce operational risk over time.

1. **[XC-7]** Instrument backend services with Prometheus counters: inventory reservation rates, payment outcomes, notification send totals, cache hit/miss rates.
2. **[XC-8]** Add trace ID propagation to outgoing gRPC metadata and Kafka message headers from all backend services.
3. **[order-management-service M-3]** Add status precondition to `MarkReserved`/`MarkCancelled` repository UPDATE clauses.
4. **[auth-service M-2]** Add minimum 8-character password length check to the `Register` handler.
5. **[api-gateway M-3]** Fix blocklist to fail closed (configurable) on Redis error; add `BLOCKLIST_FAIL_OPEN` env var.
6. **[api-gateway M-4]** Fix rate-limiter Lua script member collision — append a random suffix to the ZADD member.
7. **[api-gateway M-2]** Introduce `AdminService` interface in the api-gateway admin handler; move SQL to a repository.
8. **[currency-service M-1]** Wire `CacheHits`, `CacheMisses`, and `FetchDuration` instruments to actual call sites.
9. **[currency-service M-3]** Remove default DB password; fail fast if `DB_PASSWORD` is unset.
10. **[seller-ui M-1]** Decompose `DashboardLayout` into `<AuthGuard>`, `<Sidebar>`, `<MobileDrawer>`, `<ThemePicker>`.
11. **[seller-ui M-2]** Add lightweight JWT expiry check in `middleware.ts` using `jose` `decodeJwt` to eliminate the Splash flash.
12. **[seller-ui M-3, admin-ui H-4, backoffice-ui M-5]** Add `app/dashboard/error.tsx` per-segment error boundaries in all three UI services.
13. **[admin-ui M-1, backoffice-ui M-4]** Fix theme FOUC in all UI services with a blocking inline `<script>` in `<head>` before first paint.
14. **[backoffice-ui M-1]** Add 300ms debounce to users page search input, mirroring the products page pattern.
15. **[notification-service M-2]** Replace flat 5-second Kafka retry with exponential backoff (1s → 60s cap).
16. **[payment-service M-2]** Migrate `LedgerDebit`/`LedgerCredit` constants to UPPER_SNAKE_CASE with a DB migration.
17. **[order-management-service H-3]** Decompose the 163-line `Checkout` method into focused private helpers.
18. **[product-catalog-service M-2]** Add `ProductStatus.IsValid()` helper and call it in service-layer create/update methods.
19. **[seller-ui H-4, H-5]** Add stable `id` field to `PendingFile`; replace index-based state updates with functional updates.
20. **[buyer-ui H-2, H-3]** Consolidate `GW` constant to `lib/gateway.ts`; replace local `publicImageUrl` with import from `@/lib/images`.

---

### Backlog — Low Priority

- Move `domain.Claims` to `pkg/crypto` (auth-service L-1)
- Change `User.Role` field type from `string` to `domain.Role` (auth-service L-2)
- Add unary gRPC logging interceptor to replace per-method boilerplate (auth-service L-4)
- Use `time.RFC3339Nano` for proto timestamp fields (product-catalog-service L-1)
- Add nil UUID guard to `GetSKUByID` (product-catalog-service L-3)
- Remove `inventory-service.exe` from git and add `*.exe` to `.gitignore` (inventory-service L-2)
- Add reservation expiry sweep job (inventory-service L-3)
- Add typed `WebhookStatus` constants (payment-service L-2)
- Fix `go.mod` Go version from `1.25.0` to `1.24.x` in notification-service (notification-service L-3)
- Add `RATE_BASE_CURRENCY` env var to currency-service worker (currency-service L-4)
- Add `MigrationDSN()` method to currency-service `Config` (currency-service L-3)
- Derive `activeCategory` from `useSearchParams` in buyer-ui `Navbar` (buyer-ui L-3)
- Return discriminated union from `apiFetch` instead of `null` (buyer-ui L-4)
- Add `aria-label` to all icon-only buttons across buyer-ui, seller-ui, admin-ui (L-series accessibility)
- Replace Google Fonts CSS `@import` with `next/font/google` in admin-ui and backoffice-ui
- Fix `BulkIO.tsx` CSV header parser to use `splitCSVLine` (backoffice-ui L-2)
- Implement or remove two-key shortcut hints in `CommandPalette` (backoffice-ui L-1)
- Add `eslint-plugin-jsx-a11y` to admin-ui devDependencies
- Fix `lucide-react@^1.20.0` non-existent version range in admin-ui `package.json`

---

## Final Verdict

**CHANGES_REQUIRED**

The platform demonstrates architectural maturity in its backend service design — the layered structure, repository interfaces, transactional outbox, and idempotency patterns are all correctly implemented. However, it cannot be promoted to production in its current state for three reasons:

First, four active data-integrity bugs (order-management-service C-1, payment-service C-1, product-catalog-service C-1, payment-service H-1) can produce zombie orders, corrupted payment ledgers, or unauthorized product mutations under conditions that are not edge cases — they include DB write failures under load and concurrent checkout race conditions.

Second, two authentication bypasses exist in the seller-ui (C-1: no role check on login; C-2: unrestricted proxy), allowing any buyer account to access the seller dashboard and any gateway endpoint. Combined with the missing proxy allowlists in admin-ui and backoffice-ui, the BFF layer provides effectively no access control beyond cookie presence.

Third, the complete absence of automated tests across nine services means neither the security controls nor the data-integrity paths have been verified. The three bugs identified above would have been caught by unit tests with mock dependencies. The platform needs a minimum viable test harness before further development velocity is trusted.

The Sprint 1 items are non-negotiable before any production deployment. Sprints 2 and 3 address the remaining security hardening and observability gaps that are required for a production-grade platform. The inventory-service is the sole service with no critical findings and a reasonable security posture; it may be promoted independently once XC-1 and XC-7 are addressed.
