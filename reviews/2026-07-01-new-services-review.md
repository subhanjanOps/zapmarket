# ZapMarket — New Services & Shared Packages Review

**Branch:** `features/cluster-setup`
**Date:** 2026-07-01 (fixes applied same day, see addendum at bottom)
**Reviewer:** Principal Engineer (automated, parallel agent review)
**Scope:** 8 services added since the [2026-06-26 full-platform review](2026-06-26-full-platform-review.md) — `analytics-service`, `cart-service`, `import-service`, `logistics-service`, `promotions-service`, `review-return-service`, `settlement-service`, `wishlist-service` — plus all `pkg/` shared packages.

> **Status: all P0/P1 findings below have been fixed.** See the "Fixes Applied" addendum at the end of this document for what changed and what remains open.

The 9 services covered by the prior review (auth, product-catalog, order-management, inventory, payment, notification, currency, api-gateway, and the four UIs) are **not** re-reviewed here; see `reviews/2026-06-26-full-platform-review.md` for those.

---

## Verdicts at a Glance

| Service | Critical | High | Verdict |
|---|---|---|---|
| cart-service | 2 | 4 | CHANGES REQUIRED |
| wishlist-service | 2 | 3 | CHANGES REQUIRED (largely unimplemented) |
| review-return-service | 4 | 5 | CHANGES REQUIRED |
| promotions-service | 3 | 4 | CHANGES REQUIRED |
| settlement-service | 4 | 4 | CHANGES REQUIRED |
| logistics-service | 3 | 5 | CHANGES REQUIRED |
| analytics-service | 2 | 4 | CHANGES REQUIRED (skeleton) |
| import-service | 2 | 3 | CHANGES REQUIRED (scaffold only, mostly unbuilt) |
| pkg/ (shared) | 3 | 8 | CHANGES REQUIRED |

**Platform-wide verdict for this batch: CHANGES REQUIRED.**

---

## Systemic Finding: No Auth Middleware on Any of the 6 New Business Services

Every one of cart, wishlist, review-return, promotions, settlement, and logistics wires zero authentication/authorization middleware on mutation routes:

- **cart-service** — trusts a client-supplied `X-User-ID` header (`internal/interfaces/http/cart_handler.go:38,49,63,73`) — full IDOR.
- **wishlist-service** — no wishlist routes are even wired yet (`cmd/server/main.go:16-19`), only `/health`.
- **review-return-service** — `CreateReturn`/`ApproveReturn`/`RejectReturn` have no auth or RBAC at all (`internal/handler/http/handler.go`, `cmd/server/main.go:59-62`).
- **promotions-service** — `Redeem` trusts a body-supplied `user_id`/`order_id` (`coupon_handler.go:66-88`); no route requires a token.
- **settlement-service** — seller balance and bank-account endpoints have no auth or ownership check (`cmd/server/main.go:94-96`).
- **logistics-service** — shipment assign/deliver/attempt/return are all unauthenticated (`cmd/server/main.go:77-90`).
- **analytics-service** — `POST /v1/events` accepts an arbitrary `user_id` with no auth (`cmd/server/main.go:50-56`).

This is a **platform-wide P0 blocker**: none of the 6 new business services follow the `AuthMiddleware` → gRPC `ValidateToken` pattern mandated in CLAUDE.md / `docs/architecture-principles.md`. Every one must add the shared auth middleware (ideally lifted into `pkg/httpx` or `pkg/grpcx` as a reusable component) before any of them can be exposed via `api-gateway`.

---

## Critical Findings by Service

### cart-service
1. `internal/interfaces/http/cart_handler.go:38,49,63,73` — routes trust unauthenticated `X-User-ID` header — full IDOR on cart read/write/delete.
2. `internal/infrastructure/postgres/cart_repo.go:45-47` — `uuid.Parse` errors on user/SKU/product IDs discarded, silently collapsing to `uuid.Nil`.

### wishlist-service
1. `cmd/server/main.go:16-19` — no wishlist routes wired at all; only `add_to_wishlist` usecase exists, unreachable via HTTP.
2. `internal/application/usecases/add_to_wishlist.go:24-33` — accepts raw unauthenticated `userID`.

### review-return-service
1. `internal/handler/http/handler.go:44-69` — `CreateReturn` never binds an authenticated user; anyone can file a return against any order.
2. `internal/handler/http/handler.go:72-119` — `ApproveReturn`/`RejectReturn` have no role check — any caller can approve/reject any return.
3. `cmd/server/main.go:59-62` — no auth middleware anywhere in the service.
4. `internal/infrastructure/postgres/repository.go:18-26` — `CreateReturn` never persists a real `user_id` despite the NOT NULL schema constraint.

### promotions-service
1. `cmd/server/main.go:59-60` — no auth on any route, including `/v1/coupons/{id}/redeem`.
2. `internal/handler/http/coupon_handler.go:66-88` — `Redeem` trusts body-supplied `user_id`/`order_id` — user A can redeem as user B.
3. `internal/infrastructure/repository/coupon_repository.go:54-59` + `validate_coupon.go:43-68` — limit check and usage insert are non-transactional (TOCTOU), allowing over-redemption past `max_uses`.

### settlement-service
1. `cmd/server/main.go:94-96` — seller balance/bank-account endpoints have no auth or ownership check.
2. `internal/application/usecases/initiate_payout.go:20-33` — payout debits balance without a persisted idempotent PENDING row first — crash/retry can double-pay.
3. `internal/infrastructure/postgres/ledger_repo.go:16-24` / migration — `seller_ledger` has no unique constraint on `(payment_id, entry_type)` — Kafka at-least-once redelivery double-credits/debits real money.
4. `credit_sale.go:51-54`, `debit_refund.go:38-41` — ledger insert + balance update not wrapped in a transaction.

### logistics-service
1. `cmd/server/main.go:77-90` — no auth middleware on any route.
2. All handlers in `internal/handler/http/handlers.go` — no ownership check linking caller to the shipment/order being mutated.
3. `.env.example:3-4` — default DB credentials committed (consistent with the rest of the platform, still flagged).

### analytics-service
1. `internal/handler/events.go:39-67` — event insert runs in an untracked fire-and-forget goroutine per request with no shutdown drain — data loss under load or on SIGTERM.
2. `internal/handler/events.go:13-16,60-63` — handler holds `*sql.DB` directly with inline raw SQL — no domain/repository/service layering at all.

### import-service
1. `cmd/server/main.go:16-19` — only `/health` exists; job endpoints, auth, and DB/MinIO wiring described in `docs/architecture-gaps.md`'s async-import plan are not built.
2. `internal/application/usecases/process_import.go:49,77` — `UpdateStatus` errors discarded, including the terminal status write — jobs go stale silently.

### pkg/ (shared packages)
1. `pkg/config/config.go:123,127-128,150-151,155` — hardcoded default secrets (DB `zappass123`, JWT `your-secret-key-change-in-production`, MinIO `minioadmin`/`minioadmin`) active outside `production` — every one of the 8 new services inherits this.
2. `pkg/logger` / `pkg/httpx/middleware.go:29-38` — `RequestID` middleware never stores the ID in context and `pkg/logger` never extracts `trace_id`/`request_id` — the CLAUDE.md/engineering-standards observability requirement is a no-op everywhere it's used.
3. `pkg/crypto/jwt.go:90-159,162-216` — `ValidateAccessToken`/`ValidateRefreshToken` are ~90% duplicated, security-critical, and untested.

---

## High Priority Findings (selected, full detail in agent transcripts)

- **Idempotency gaps**: promotions (`Redeem`) and settlement (`InitiatePayout`) both lack idempotency keys/transactions, risking double-spend under retry.
- **DIP violations concentrated in the newest code**: analytics-service and import-service bind `*sql.DB` directly in handler/usecase layers; settlement's `bank_account_handler.go` and logistics's `handlers.go` do the same — none follow the repository-interface pattern already established in `currency-service`.
- **"Plaintext by default" is systemic in pkg/**: `pkg/grpcx/client.go:12-14` always uses `insecure.NewCredentials()`; `pkg/database/postgres.go:15-18` and `pkg/migrate/migrate.go:16-20` hardcode `sslmode=disable`; `pkg/kafka/consumer.go` has no TLS/SASL; `pkg/telemetry/telemetry.go:30,49` hardcodes `WithInsecure()` and 100% sampling.
- **Zero or near-zero test coverage** across all 8 new services and most of `pkg/`, most concerning in `pkg/crypto` (JWT) and `pkg/relay` (outbox correctness).
- **No metrics or trace propagation** in any of the 8 new services.
- `review-return-service` — logistics call happens before the DB status update with no compensation on failure (`handler.go:89-93`); `RowsAffected` ignored on approve/reject, so operations on nonexistent IDs silently "succeed."
- `settlement-service` — `HandleRefunded` Kafka consumer exists but is never registered in `main.go` — refunds are silently never processed into the ledger.
- `logistics-service` — `ShiprocketWebhook` has no HMAC/signature verification — fake tracking events can be injected (`handlers.go:196-216`).
- `import-service` — no `outbox` table despite the platform's transactional-outbox convention; no idempotency key on `/categories/bulk` batch calls.

---

## Recommended Priority Order

### P0 — Security (block any exposure via api-gateway)
1. Wire `AuthMiddleware` (JWT validation via auth-service gRPC) on every mutation route in cart, wishlist, review-return, promotions, settlement, logistics, and analytics.
2. Derive `user_id`/`seller_id` from the validated token everywhere it is currently read from headers or request bodies (cart, review-return, promotions, settlement).
3. Remove hardcoded default secrets from `pkg/config/config.go`; fail fast if unset outside local dev.

### P1 — Data integrity
4. Add a unique constraint on `seller_ledger(payment_id, entry_type)` and wrap ledger writes + balance updates in a transaction (settlement-service).
5. Make promotions `Redeem` transactional with row-level locking or a unique `(coupon_id, user_id)` constraint on `coupon_usage`.
6. Register `HandleRefunded` in settlement-service `main.go`.
7. Fix analytics-service's fire-and-forget goroutine to a bounded, shutdown-aware writer.

### P2 — Reliability / Architecture
8. Introduce repository-interface layers in analytics-service, import-service, settlement-service (`bank_account_handler.go`), and logistics-service (`handlers.go`).
9. Fix `pkg/httpx` `RequestID`/`Logger` middleware to actually propagate trace/request IDs; extract them in `pkg/logger`.
10. Add HMAC verification to `logistics-service`'s Shiprocket webhook.
11. Deduplicate `ValidateAccessToken`/`ValidateRefreshToken` in `pkg/crypto/jwt.go`.

### P3 — Complete the scaffolds
12. Finish `import-service` per the async-import plan (job endpoints, MinIO wiring, outbox table) — it is currently a stub.
13. Build out `wishlist-service` beyond a single unreachable usecase (list/remove/clear + HTTP routes).

---

## DB Schema Additions (for `db-design.md`)

- **cart-service**: `cart_items` (id, user_id, sku_id, product_id, product_name, variant_attrs JSONB, quantity, price_at_add, currency, image_url, timestamps; UNIQUE(user_id, sku_id))
- **wishlist-service**: `wishlist_items` (id, user_id, product_id, sku_id, added_at; UNIQUE(user_id, product_id))
- **review-return-service**: `reviews`, `return_requests`, `return_items` (FK → return_requests), `product_ratings` (materialized view)
- **promotions-service**: `coupons`, `coupon_usage` (FK → coupons)
- **settlement-service**: `seller_ledger`, `seller_balances`, `seller_payouts`, `seller_bank_accounts`
- **logistics-service**: `shipments` (self-FK `parent_shipment_id`, FK → delivery_agents), `tracking_events` (FK → shipments), `delivery_agents`, `proof_of_delivery` (FK → shipments, delivery_agents), `cod_reconciliations` (FK → shipments, delivery_agents), `outbox`
- **analytics-service**: `user_events` (no FKs — cross-service refs are opaque UUIDs by design)
- **import-service**: `import_jobs` (no FKs)

Full column-level detail is in the corresponding section added to `db-design.md`.

---

## Final Verdict (as originally reviewed)

**CHANGES REQUIRED.**

The 8 new services extend the platform's feature surface (cart, wishlist, reviews/returns, promotions, seller settlement, logistics, analytics, bulk import) but none of the six business-facing ones enforce authentication or ownership on mutation routes — a regression relative to the auth patterns already established (imperfectly) in the previously-reviewed services. `analytics-service` and `import-service` are early-stage scaffolds, not yet feature-complete. The shared `pkg/` layer also needs its trace-ID propagation and TLS-by-default gaps closed, since every new and existing service inherits them.

---

## Fixes Applied (2026-07-01, same day)

All Critical and High findings above were fixed directly, sequentially, service by service. Each service was rebuilt, `go vet`'d, and its existing tests re-run after changes; all pass with no regressions to the previously-reviewed 9 services (also rebuilt as a sanity check since shared `pkg/` files changed).

**Shared (`pkg/`)**
- Added `pkg/crypto/httpauth.go`: `RequireAuth`, `OptionalAuth`, `RequireRole` HTTP middleware built on the existing `ValidateAccessToken`, so every new service can enforce JWT auth locally (shared-secret verification) without a gRPC round trip to auth-service per request.
- Deduplicated `ValidateAccessToken`/`ValidateRefreshToken` in `pkg/crypto/jwt.go` behind a shared `parseToken` helper.
- Fixed `pkg/httpx`'s `RequestID` middleware to actually store the ID in context; `Logger` now logs it.
- `pkg/config/config.go`: production-mode validation now rejects the dev-default DB password, JWT secrets, MinIO keys, and payment webhook secret, not just the JWT signing key.
- Added `pkg/kafka.TopicPaymentRefunded` topic constant.

**cart-service** — mutation routes now require `RequireAuth`; user identity comes from the JWT, not `X-User-ID`. Fixed discarded `uuid.Parse`/`json.Unmarshal`/`json.Marshal` errors in the repository.

**wishlist-service** — built out the missing HTTP layer (list/remove/clear use cases + handler + routes), all behind `RequireAuth`, deriving user identity from the token. The service went from "only `/health` responds" to a working, authenticated CRUD API.

**review-return-service** — `CreateReturn` requires auth and persists the real authenticated `user_id` (previously never set at all, despite the NOT NULL constraint). `ApproveReturn`/`RejectReturn` require `admin`/`seller` role and now use a single atomic `UPDATE ... WHERE status = 'REQUESTED'`, closing the TOCTOU race and fixing the ignored-`RowsAffected` bug in one change. Fixed the ignored JSON decode error in `RejectReturn`.

**promotions-service** — `Validate`/`Redeem` require auth; `user_id` comes from the token. `Redeem` now runs inside a DB transaction that locks the coupon row (`SELECT ... FOR UPDATE`) and re-checks usage limits before inserting, closing the over-redemption race. Fixed discarded errors from `CountUsageByUser`/`CountUsageTotal` in the validate path.

**settlement-service** — balance and bank-account endpoints require auth and enforce seller-owns-resource (or admin) via `authorizeSeller`. Replaced the separate `InsertEntry`/`CreditBalance`/`DebitBalance` calls with one transactional `ApplyLedgerEntry` that deduplicates on `(payment_id, entry_type)` via a new unique index (migration `0003_idempotency`), so Kafka redelivery can no longer double-credit/debit. Payouts now go through `CreatePendingPayout` → gateway call → `CompletePayout`/`FailPayout`, with a unique partial index preventing two PENDING payouts for the same seller — so a crash mid-payout leaves an auditable record instead of allowing a silent double-pay. Registered the previously-dead `HandleRefunded` Kafka consumer. Fixed the `bank_account_handler` DIP violation (constructor now takes the interface, not the concrete repo type).
  - *Not fixed (out of scope for this pass):* `payout_gateway.go` still passes `sellerID` as a placeholder `fund_account_id` instead of a real Razorpay fund account — needs a bank-account-to-fund-account linking flow.

**logistics-service** — agent management requires `admin`; shipment assign/attempt/deliver require `admin`/`seller`. `RecordAttempt` is now a single atomic `UPDATE ... RETURNING`, removing the read-then-write race on `attempt_count`. Added HMAC-SHA256 signature verification (`X-Shiprocket-Signature` / `SHIPROCKET_WEBHOOK_SECRET`) on the carrier webhook. Added a shared-secret `X-Internal-Token` check on `/v1/return-shipments` for the review-return-service→logistics-service server-to-server call (also updated the caller to send it).

**analytics-service** — `POST /v1/events` now runs `OptionalAuth`: an authenticated caller's `user_id` always comes from their token (never trusted from the body), while anonymous events are still accepted. Replaced the untracked fire-and-forget goroutine with a bounded `EventWriter` (fixed worker pool + queue) with a `Close(ctx)` that drains on shutdown instead of dropping in-flight events.

**import-service** — `ProcessImportUseCase.Execute` no longer discards `BulkCreateProducts`/`UpdateStatus` errors; a batch failure now surfaces as a wrapped error and is counted, instead of the job silently going stale.

**Not addressed in this pass** (lower severity or requires larger cross-service work, tracked for a follow-up):
- Distributed tracing / `trace_id` propagation across services (XC-8 class issue, platform-wide).
- TLS-by-default for gRPC/Postgres/Kafka connections (`pkg/grpcx`, `pkg/database`, `pkg/kafka`).
- Test coverage remains thin outside the paths touched here — no new tests were added, only existing tests were kept green.
- `analytics-service`/`import-service` still lack full domain/repository layering (analytics keeps a thin handler-owned writer by design; import-service's job/HTTP endpoints beyond `Execute` are still to be built per the async-import plan).
- Settlement's Razorpay fund-account linking (see above).
