# ZapMarket — Full System Code Review

**Branch:** `features/cluster-setup`  
**Date:** 2026-06-22  
**Reviewer:** Principal Engineer (automated, Opus model)  
**Scope:** All 9 services, use-case flow order

---

## Verdicts at a Glance

| Service | Verdict | Blockers |
|---|---|---|
| `auth-service` | **CHANGES REQUIRED** | 3 critical |
| `product-catalog-service` | **CHANGES REQUIRED** | 2 critical |
| `order-management-service` | **CHANGES REQUIRED** | 2 critical |
| `payment-service` | **CHANGES REQUIRED** | 1 critical |
| `inventory-service` | **CHANGES REQUIRED** | 2 critical |
| `notification-service` | **CHANGES REQUIRED** | 2 critical |
| `api-gateway` | **CHANGES REQUIRED** | 3 critical |
| `currency-service` | **APPROVED WITH RECOMMENDATIONS** | 0 critical |
| `seller-ui` | **APPROVED WITH RECOMMENDATIONS** | 0 critical (2 merge blockers) |

---

## Critical Findings — Must Fix Before Production

### auth-service (`reviews/auth-service.md`)
- **C1** Refresh token rotation missing; revocation untested — a stolen refresh token grants indefinite access
- **C2** `TokenHash` is overwritten with the raw token (secret storage footgun)
- **C3** OAuth `state` is derived from `time.Now()` and never validated — CSRF on OAuth callback

### product-catalog-service (`reviews/product-catalog-service.md`)
- **C1** No seller ownership enforcement — Seller A can update/delete Seller B's products and SKUs (IDOR / horizontal privilege escalation)
- **C2** Image mutations ignore `product_id` path param — any image can be manipulated via any product URL

### api-gateway (`reviews/api-gateway.md`)
- **C1** Client-supplied `X-User-Role` header is never stripped on `auth_mode: none` routes — any client can forge admin identity to downstream services
- **C2** Data race: `upstreams` map is read/written across goroutines without a lock
- **C3** CORS: origin is reflected without validation; `EXTRA_ALLOWED_ORIGINS` unchecked

### order-management-service (`reviews/order-management-service.md`)
- **C1** `unit_price` is taken from the buyer's checkout request without validating against product-catalog-service — buyers can self-issue arbitrary prices
- **C2** `break` inside `select` in the idempotency spin-loop does not exit the outer `for` — goroutine spins for the full 10-second lock TTL

### payment-service (`reviews/payment-service.md`)
- **C1** Zero test coverage on a service that writes immutable money ledger rows

### inventory-service (`reviews/inventory-service.md`)
- **C1** Redis counter can drift permanently below the DB value when the Postgres compensation `IncrBy` also fails — stock becomes unreservable indefinitely
- **C2** No expired-reservation sweep job — `qty_reserved` never decrements if order-management-service fails, eventually converging available stock to zero

### notification-service (`reviews/notification-service.md`)
- **C1** `formatAmount` divides JPY/KRW/IDR by 100 even though those are zero-decimal currencies — ¥1,500 displays as ¥15
- **C2** Redis dedup `SetNX` is written *before* `notifier.Send`; a transient Send failure permanently drops the notification on retry

---

## High-Priority Findings (selected cross-cutting)

| # | Service | Finding |
|---|---|---|
| H1 | auth-service | `ValidateToken` does a DB read on every gRPC call — auth DB is the fleet throughput ceiling; no caching |
| H1 | api-gateway | Rate limiter runs before auth; `UserFromContext` is always nil — per-user rate-limit tier is dead code |
| H1 | api-gateway | Audit drain uses the cancelled context on shutdown — final audit entries are silently lost |
| H1 | product-catalog-service | Zero test files despite mockgen seams being wired |
| H1 | notification-service | `CommitInterval: time.Second` races with manual `CommitMessages` — can commit offsets before successful processing |
| H2 | notification-service | Dedup TTL of 1 hour is too short; Kafka can redeliver days later (should be 72h) |
| H2 | payment-service | Post-charge crash leaves money captured at gateway but DB in PENDING with no compensating refund |
| H3 | payment-service | `ChargeCard` returns 409 under lock contention; order-management retries — incompatible strategies can cause double-charge |
| H4 | payment-service | `RefundPayment` has no idempotency — duplicate calls cause double refunds |
| H1 | inventory-service | gRPC server has no authentication — any in-cluster service can manipulate stock |
| H4 | inventory-service | Redis stock keys have no TTL (`0`) — drift from any cause is permanent |
| H1 | currency-service | `ToggleCurrency` trusts spoofable `X-User-Role` header; no defense-in-depth |
| H3 | currency-service | Worker goroutine is never awaited on SIGTERM — in-flight ingest abandoned |

---

## Merge Blockers in seller-ui

- **C1** Inconsistent proxy path prefix: currency/list use `/api/proxy/api/v1/...` but preferences use `/api/proxy/v1/...`; one convention is wrong and errors are silently swallowed in `currency.tsx`
- **C2** `app/api/fx-rates/route.ts` is staged in git but missing from disk — ghost file; run `git rm --cached services/seller-ui/app/api/fx-rates/route.ts`

---

## Systemic Patterns

1. **No tests on financial paths** — product-catalog-service, payment-service, inventory-service, notification-service handler, and seller-ui all have zero test files. The highest-risk code has zero coverage.
2. **gRPC without service-to-service auth** — payment-service and inventory-service gRPC servers accept connections from any in-cluster caller. mTLS or a shared secret header is needed.
3. **Header injection surface** — both api-gateway and currency-service trust caller-supplied identity headers. The gateway must strip `X-User-*` headers on inbound requests before auth runs.
4. **No distributed tracing** — none of the services propagate a `trace-id`/`request-id`. Debugging across the call chain (auth → product-catalog → order → payment → inventory → notification) is currently impossible in production.
5. **Observability gap** — most services return 200 on health checks regardless of downstream (Redis, DB) status; structured logging lacks correlation IDs; metrics are present only in currency-service.
6. **Shutdown correctness** — multiple services abandon in-flight goroutines on SIGTERM (currency-service worker, api-gateway audit writer, gRPC client connections never closed in order-management-service).

---

## Recommended Priority Order

### P0 — Security (block ship)
1. Strip `X-User-*` headers from inbound requests in api-gateway before auth runs (C1 gateway)
2. Add seller ownership check on all product/SKU/image mutations (C1 product-catalog)
3. Fix refresh token rotation and add rotation test (C1 auth)

### P1 — Data integrity (block ship)
4. Validate `unit_price` against product-catalog in checkout (C1 order-management)
5. Fix `break` in spin-loop (C2 order-management)
6. Move Redis dedup `SetNX` to after successful `notifier.Send` (C2 notification)
7. Fix `formatAmount` zero-decimal division (C1 notification)
8. Add expired-reservation sweep job (C2 inventory)

### P2 — Reliability
9. Write payment-service tests; add `MarkCaptured` status guard; add `RefundPayment` idempotency
10. Fix inventory Redis drift compensation; add TTL to stock keys
11. Fix api-gateway data race on `upstreams` map
12. Fix api-gateway audit drain context on shutdown
13. Await currency-service worker goroutine on shutdown

### P3 — Quality
14. Add `ValidateToken` caching in auth-service (in-memory LRU, 30s TTL)
15. Extract `formatAmount` from notification handler.go into `internal/currency/`
16. Add service-to-service auth to payment-service and inventory-service gRPC servers
17. Add distributed trace ID propagation across all services
18. Deduplicate `toCents`/`fromCents` in seller-ui into a shared lib

---

## Individual Review Files

- [`auth-service.md`](auth-service.md)
- [`product-catalog-service.md`](product-catalog-service.md)
- [`order-management-service.md`](order-management-service.md)
- [`payment-service.md`](payment-service.md)
- [`inventory-service.md`](inventory-service.md)
- [`notification-service.md`](notification-service.md)
- [`api-gateway.md`](api-gateway.md)
- [`currency-service.md`](currency-service.md)
- [`seller-ui.md`](seller-ui.md)
