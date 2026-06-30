# ZapMarket vs Amazon India — Comparison Report

**Date:** 2026-06-30 | **Branch:** features/cluster-setup

---

## Executive Summary

**Benchmark: Amazon India** — chosen because zapmarket's stack signals India-market targeting (Razorpay schema, GSTIN/PAN in seller profiles, pincode zones in inventory, INR-denominated payments in paise). Flipkart is equivalent but Amazon's architecture is better documented for gap analysis.

### 3 Strengths
1. **Solid event-driven core** — choreography saga (order→inventory→payment), transactional outbox per service, and DB-level idempotency keys are architecturally correct choices that scale beyond a monolith.
2. **Auth depth** — OTP verification, social OAuth (Google + Facebook), password reset via token and OTP, Redis-backed JWT blacklisting, and IP rate limiting on OTP routes are all implemented.
3. **Full-stack surface** — 15+ services covering catalog, cart, orders, payments, inventory, logistics, promotions, reviews, settlements, currency, import, and 4 UIs deployed from a single compose file with Prometheus + Grafana + Loki observability wired.

### 5 Critical Gaps
1. **Kafka/Debezium not active** — saga events never leave the outbox; all cross-service consistency is broken at runtime.
2. **No saga compensations** — payment failure does not release inventory reservations; TTL expiry is the only fallback.
3. **No CI/CD** — zero automated build, test, or deploy pipelines; no k8s manifests.
4. **Promotions not wired into checkout** — coupon/discount logic exists in promotions-service but is never called during order creation or payment.
5. **No account lockout or MFA** — login endpoints are unrate-limited; no TOTP/2FA; no device session management.

---

## Service Inventory

| Service | Lang | Role | Talks To | DB |
|---|---|---|---|---|
| auth-service | Go | Identity, JWT, OAuth | (gRPC server) | `userauth` (PG) |
| product-catalog-service | Go | Products, categories, search | auth-service gRPC, Typesense | `productcatalog` (PG) |
| order-management-service | Go | Order lifecycle, saga orchestration | catalog/inventory/payment gRPC, Kafka | `orders` (PG) |
| payment-service | Go | Stripe gateway, refunds, ledger | Kafka consumer, auth gRPC | `payments` (PG) |
| inventory-service | Go | Stock reservation, pincode zones | Kafka consumer, gRPC server | `inventory` (PG) + Redis |
| cart-service | Go | Cart persistence | catalog gRPC | `cart` (PG) |
| currency-service | Go | FX rates, multi-currency | Frankfurter/OpenExchangeRates HTTP | `currency` (PG) |
| notification-service | Go | Email/SMS dispatch | Kafka consumer, Twilio/Resend | — |
| settlement-service | Go | Seller payouts | — | `settlement` (PG) |
| logistics-service | Go | Shipment tracking | — | `logistics` (PG) |
| promotions-service | Go | Coupons, discounts | — | `promotions` (PG) |
| review-return-service | Go | Reviews, returns | — | `reviews` (PG) |
| wishlist-service | Go | Buyer wishlists | — | `wishlist` (PG) |
| import-service | Go | Bulk product CSV import | product-catalog HTTP | `import` (PG) + MinIO |
| api-gateway | Go | Reverse proxy, route registry | All services HTTP | `gateway` (PG) |
| buyer-ui | Next.js 15 | Buyer storefront | api-gateway | — |
| seller-ui | Next.js 16 | Seller dashboard | api-gateway | — |
| admin-ui | Next.js 16 | Admin panel | api-gateway | — |
| backoffice-ui | Next.js 16 | Internal ops | api-gateway | — |

---

## 1. Architecture and Scalability

ZapMarket correctly uses separate DB-per-service and event-driven saga coordination — the right foundation. **First bottleneck: the API gateway.** It proxies all traffic through a single Go process backed by a PostgreSQL route registry (5 migrations worth of routing rules). Under load, route lookups hit the DB on every request with no caching visible in the gateway code. Amazon uses edge-based routing (CloudFront + ALB) that never touches a database per request.

**Second bottleneck: Redis as hard dependency for inventory.** Inventory-service exits on Redis unavailability with no PostgreSQL fallback. One Redis restart takes down all stock reservation — a single point of failure that Amazon eliminates with regional Redis clusters and fallback read paths.

**Fix:** Add an in-process LRU cache for gateway route table (refresh every 60s); add `SELECT FOR UPDATE` fallback in inventory-service when Redis is unreachable.

---

## 2. Data and Consistency

| Aspect | ZapMarket | Amazon India |
|---|---|---|
| Cross-service consistency | Transactional outbox + choreography saga | Distributed saga + compensating transactions |
| Outbox drain | **Not active** (Debezium deferred to Stage 7) | CDC active, sub-second lag |
| Saga compensations | TTL expiry only (no programmatic rollback) | Full compensating transactions per saga step |
| Inventory locking | Redis Lua + PG ledger | Redis + DynamoDB conditional writes |
| Idempotency | DB UNIQUE key + Redis cache | Idempotency tokens throughout |
| Reservation cleanup | No sweep job (TTL column, no executor) | Background sweeper per region |

**Fix:** Enable Kafka + Debezium in Stage 7 now (it is the load-bearing gap); add one compensation handler that publishes `reservation.release` when `payment.failed` is consumed by inventory-service.

---

## 3. Feature/UX Parity

| Feature | ZapMarket | Amazon India |
|---|---|---|
| Product catalog + search | ✅ Typesense | ✅ Elasticsearch + ML ranking |
| Multi-currency | ✅ Live FX rates | ✅ |
| Cart persistence | ✅ DB-backed | ✅ |
| Guest checkout | ❌ | ✅ |
| Saved payment methods / wallet | ❌ | ✅ Amazon Pay |
| Coupons wired into checkout | ❌ (exists, not connected) | ✅ |
| Tax calculation | ❌ Schema only, no logic | ✅ GST automated |
| Shipping address on orders | ❌ No address column in orders table | ✅ |
| Seller approval workflow | ❌ No status field | ✅ Seller verification |
| Reviews and returns | ✅ Service exists | ✅ |
| Wishlists | ✅ | ✅ |
| Notifications (email/SMS) | ✅ Twilio + Resend | ✅ |
| Logistics tracking | ✅ Service exists | ✅ |
| Bulk product import | ✅ CSV via MinIO | ✅ |
| Promotions / flash sales | ✅ Service exists | ✅ |
| MFA at login | ❌ | ✅ OTP required |
| Phone-OTP login (passwordless) | ❌ | ✅ |

---

## 4. Security

| Area | ZapMarket | Amazon India |
|---|---|---|
| JWT auth | ✅ RS256, refresh tokens, Redis blacklist | ✅ |
| Social OAuth | ✅ Google + Facebook | ✅ + Apple |
| OTP verification | ✅ Phone + email | ✅ |
| MFA at login | ❌ No TOTP/HOTP | ✅ |
| Login rate limiting | ❌ `/auth/login` unprotected | ✅ |
| Account lockout | ❌ No failed-attempt counter | ✅ |
| Device/session tracking | ❌ No device fingerprint on tokens | ✅ |
| Secrets management | ❌ Env vars in compose | ✅ Secrets Manager |
| Stripe webhook sig validation | ✅ (endpoint wired) | ✅ |
| Seller GSTIN/PAN storage | ✅ | ✅ |
| Data at rest encryption | ❌ Not configured | ✅ |

**Fix:** Add `failed_login_attempts` + `locked_until` columns to `users` table; apply `IPRateLimit` middleware to `/auth/login` and `/auth/register` (same middleware already used on OTP routes — one-line addition per route).

---

## 5. Reliability and Ops

| Area | ZapMarket | Amazon India |
|---|---|---|
| Observability | Prometheus + Grafana + Loki (configured) | Full APM + distributed tracing |
| Alerting | ❌ No alert rules in Prometheus | ✅ |
| CI/CD | ❌ None | ✅ Full pipeline |
| Container orchestration | Docker Compose only | ✅ EKS |
| Connection pooling | ✅ PgBouncer configured | ✅ |
| Health checks in compose | Partial (not all services) | ✅ |
| Secrets management | ❌ Env vars | ✅ |
| Service discovery | Hardcoded container names | ✅ Dynamic |

**Fix:** Add a GitHub Actions workflow with `go test ./...` per service on PR — zero-dependency CI that catches regressions before merge.

---

## Prioritized Roadmap

| Change | Effort | Impact |
|---|---|---|
| Activate Kafka + Debezium (Stage 7) | M | H |
| Add inventory saga compensation on `payment.failed` | S | H |
| Wire promotions-service into order creation checkout call | S | H |
| Add `failed_login_attempts` + lockout + rate-limit login endpoint | S | H |
| Add `delivery_address` columns to orders table | S | H |
| Add `reservation_expiry_sweeper` scheduled job | S | M |
| Implement Razorpay gateway (schema already declares it) | M | H |
| Add shipping address binding at checkout | M | H |
| Add tax calculation logic (schema exists) | M | H |
| Add seller approval status field + admin endpoint | S | M |
| Add TOTP/MFA at login | M | M |
| Add Redis fallback (PG `qty_available`) for inventory | M | M |
| Create GitHub Actions CI pipeline | S | M |
| Add Prometheus alerting rules | S | M |
| Implement saved payment methods / wallet | L | M |
| Unify frontend to single Next.js version + shared component package | L | M |
| Add form validation (react-hook-form + zod) to all UIs | M | M |
| Produce k8s/Helm manifests for cloud deployment | L | H |
