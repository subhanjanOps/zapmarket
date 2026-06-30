# ZapMarket vs Amazon India — Comprehensive Comparison Report

**Date:** 2026-06-30 | **Branch:** features/cluster-setup  
**Benchmark:** Amazon India — chosen because ZapMarket's stack signals India-market targeting (Razorpay schema, GSTIN/PAN in seller profiles, pincode zones in inventory, paise-denominated payments).

---

## Executive Summary

### 3 Strengths
1. **Architecturally sound event core** — transactional outbox + choreography saga + DB-level idempotency keys are the right patterns; no monolith debt to pay later.
2. **Auth depth for India** — OTP via SMS/email, social OAuth (Google + Facebook), phone-linked accounts, Redis-backed JWT blacklisting, and IP rate limiting on OTP routes all implemented.
3. **Broad service surface** — 15 backend services, 4 UIs, Prometheus + Grafana + Loki, PgBouncer, and Typesense-backed search with faceting all wired in a single compose file. Impressive coverage for the stage.

### 7 Critical Gaps (expanded from 5)
1. **Kafka/Debezium not active** — saga events sit in outbox tables forever; every cross-service flow (order→inventory→payment) is broken at runtime.
2. **Last-mile / delivery agent layer is a skeleton** — no agent assignment, no COD, no proof of delivery, no 3PL adapter, no reattempt logic. The logistics-service is schema + one stub. This is the largest single gap for India operations.
3. **COD entirely absent** — no COD payment method in payment schema, order domain, or UI. COD accounts for ~60% of Indian e-commerce volume (Amazon India, Meesho, Flipkart all mandate it).
4. **No saga compensation handlers** — payment failure does not programmatically release inventory; TTL expiry is the only fallback.
5. **Promotions not wired into checkout** — full coupon/discount system built in promotions-service but never called during order creation or payment.
6. **No CI/CD, no k8s** — zero automated pipelines; no cloud-native deployment path.
7. **Seller compliance missing** — no TDS deduction, no GST on commission, no bank/UPI payout details for sellers, no KYC onboarding flow; sellers cannot be paid out legally.

---

## Service Inventory

| Service | Lang | Role | Talks To | DB |
|---|---|---|---|---|
| auth-service | Go | Identity, JWT, OAuth, OTP | (gRPC server) | `userauth` (PG) |
| product-catalog-service | Go | Products, categories, SKUs, Typesense | auth gRPC | `productcatalog` (PG) + Typesense |
| order-management-service | Go | Order lifecycle, saga orchestration | catalog/inventory/payment gRPC, Kafka | `orders` (PG) |
| payment-service | Go | Stripe gateway, refunds, ledger, webhooks | Kafka, auth gRPC | `payments` (PG) |
| inventory-service | Go | Stock reservation, pincode zones | Kafka, gRPC server | `inventory` (PG) + Redis |
| cart-service | Go | Cart persistence | catalog gRPC | `cart` (PG) |
| currency-service | Go | FX rates, multi-currency | Frankfurter/OpenExchangeRates HTTP | `currency` (PG) |
| notification-service | Go | Email/SMS dispatch | Kafka consumer, Twilio/Resend | — |
| settlement-service | Go | Seller ledger, commission deduction | — | `settlement` (PG) |
| logistics-service | Go | Shipment + tracking schema (stub) | CarrierClient interface (stub) | `logistics` (PG) |
| promotions-service | Go | Coupons (PERCENT/FIXED), usage limits | — | `promotions` (PG) |
| review-return-service | Go | Reviews (moderated), return requests | — | `reviews` (PG) |
| wishlist-service | Go | Buyer wishlists | — | `wishlist` (PG) |
| import-service | Go | Bulk CSV product import | product-catalog HTTP, MinIO | `import` (PG) |
| api-gateway | Go | Reverse proxy, DB-backed route registry | All services HTTP | `gateway` (PG) |
| buyer-ui | Next.js 15 | Buyer storefront | api-gateway | — |
| seller-ui | Next.js 16 | Seller dashboard | api-gateway | — |
| admin-ui | Next.js 16 | Admin panel | api-gateway | — |
| backoffice-ui | Next.js 16 | Internal ops | api-gateway | — |

---

## 1. Architecture and Scalability

ZapMarket correctly separates DB-per-service and uses event-driven saga coordination — the right foundation. Two architectural bottlenecks will surface first under load:

**API gateway bottleneck:** All traffic proxies through a single Go process backed by a PostgreSQL route table (5 migrations of routing rules, no cache visible in gateway code). Every request hits the DB for route resolution. Amazon uses CloudFront + ALB — zero per-request DB cost.  
**Fix:** Add in-process LRU cache for route table, refresh every 60s.

**Redis as hard inventory dependency:** Inventory-service exits on Redis unavailability with no PostgreSQL fallback. One Redis node restart takes down all stock reservation.  
**Fix:** Add `SELECT FOR UPDATE` on `qty_available` as fallback when Redis is unreachable.

| Dimension | ZapMarket | Amazon India |
|---|---|---|
| Service decomposition | ✅ 15 independent services | ✅ 100s of services |
| API routing | DB-per-request, no cache | Edge (CloudFront + ALB) |
| Async events | Outbox written, never drained | Kafka active, sub-second lag |
| Inventory concurrency | Redis Lua + PG ledger | Redis + DynamoDB conditional writes |
| Connection pooling | ✅ PgBouncer configured | ✅ RDS Proxy |

---

## 2. Last-Mile / Delivery Agent Perspective ⚠️ Critical Gap

This is the most underdeveloped area relative to what an Indian marketplace requires.

| Capability | ZapMarket | Amazon India / Flipkart |
|---|---|---|
| Shipment + tracking event schema | ✅ Tables exist | ✅ |
| 3PL courier integration | ❌ Interface stub only (no Shiprocket/Delhivery/BlueDart HTTP adapter) | ✅ Multi-courier with auto-allocation |
| Delivery agent assignment | ❌ No `delivery_agents` table, no assignment column | ✅ Agent app + assignment engine |
| Proof of delivery (OTP/signature/photo) | ❌ No field exists anywhere | ✅ OTP + photo mandatory |
| Failed delivery attempt + reattempt | ❌ No attempt counter, no rescheduling | ✅ Up to 3 attempts, buyer notified |
| Delivery slot selection by buyer | ❌ Not in order or shipment schema | ✅ Amazon 2-hour slots |
| COD support | ❌ Not a valid `gateway` value | ✅ ~60% of volume |
| COD reconciliation | ❌ | ✅ Daily agent cash handover |
| Route optimization / manifest | ❌ No route or batch logic | ✅ ML-based routing |
| Reverse logistics (return pickup) | ❌ No pickup scheduling or return shipment | ✅ |
| Hyperlocal vs intercity distinction | ❌ No delivery type field | ✅ |
| Real-time tracking feed to buyer | ❌ Schema only, no webhook ingestion | ✅ |
| Delivery agent portal / app | ❌ No agent-facing UI or API | ✅ Amazon Flex app |

**What's needed to get to V1 of last-mile:**
- Add `delivery_agents` table + assignment FK on `shipments`
- Add `proof_of_delivery` table (otp_verified, photo_url, timestamp)
- Add `attempt_count` + `next_attempt_at` on `shipments`
- Wire one real 3PL HTTP adapter (Shiprocket has a REST API and is the standard India first-mile)
- Add `COD` to payment gateway enum and a cash-on-delivery reconciliation ledger

---

## 3. Cash on Delivery (COD) — Absent End-to-End

COD is not a cosmetic feature in India — it is the dominant payment method for tier-2/3 cities and first-time buyers. Its absence means ZapMarket cannot serve a majority of the Indian addressable market.

**Impact chain:** No COD gateway value → no COD order creation → no COD in checkout UI → no COD reconciliation in settlement → no cash handover from delivery agents.

**Fix:** Add `COD` to payment `gateway` enum; create `cod_reconciliations` table linking shipment delivery confirmation to payment status; update order creation to skip Stripe/Razorpay call for COD orders and mark payment `PENDING` until delivery confirmed.

---

## 4. Data and Consistency

| Aspect | ZapMarket | Amazon India |
|---|---|---|
| Cross-service pattern | Transactional outbox + choreography saga | Saga + active CDC |
| Outbox drain | **Not active** (Debezium deferred) | Sub-second lag |
| Saga compensations | TTL expiry only | Full compensating transactions |
| Inventory locking | Redis Lua + PG ledger | Redis + DynamoDB conditional writes |
| Idempotency | ✅ DB UNIQUE + Redis cache | ✅ |
| Reservation cleanup sweeper | ❌ No scheduled job | ✅ |
| Return debit from seller settlement | ❌ Not linked | ✅ |

**Key runtime risk:** All saga steps are conditional on Kafka being active. Until Debezium is enabled, every order lands in a permanent `PENDING` state — inventory reserved, payment never charged, seller never notified.

---

## 5. Payments and Checkout

| Feature | ZapMarket | Amazon India |
|---|---|---|
| Stripe integration | ✅ Full (charge, refund, webhook) | ✅ |
| Razorpay integration | ❌ Schema declared, no Go code | ✅ Primary gateway |
| COD | ❌ | ✅ |
| Idempotency on charge | ✅ Redis NX + DB UNIQUE | ✅ |
| Refunds (partial) | ✅ `refunds` table, `PARTIALLY_REFUNDED` status | ✅ |
| Webhooks | ✅ `/webhooks/stripe` handler wired | ✅ |
| Saved payment methods / wallet | ❌ | ✅ Amazon Pay |
| Coupons wired into checkout | ❌ Promotions-service not called | ✅ |
| Delivery address on orders | ❌ No address columns in `orders` | ✅ |
| Tax (GST) calculation | ❌ Schema exists, no logic | ✅ Automated GST |
| EMI / Buy Now Pay Later | ❌ | ✅ |

---

## 6. Seller Experience and Settlement

| Feature | ZapMarket | Amazon India / Flipkart |
|---|---|---|
| Commission deduction | ✅ BPS-based, ledger entry | ✅ |
| TDS deduction | ❌ | ✅ (mandatory in India) |
| GST on commission | ❌ | ✅ (mandatory in India) |
| Payout bank/UPI details | ❌ No bank account storage | ✅ |
| Payout scheduler | ❌ Comment in code, not wired | ✅ Weekly/T+7 cycles |
| Return debit from seller payout | ❌ | ✅ |
| Seller KYC onboarding (GSTIN/PAN/bank) | ❌ Profile stored but no approval state | ✅ Multi-step verification |
| Seller approval/rejection by admin | ❌ No status field in `seller_profiles` | ✅ |
| SKU + variant attributes | ✅ JSON variant attrs, gRPC wired | ✅ |
| Seller analytics dashboard | ❌ StatCard only, no earnings/returns data | ✅ |
| Bulk product import (CSV) | ✅ Via import-service + MinIO | ✅ |

**Compliance blocker:** TDS (1% on marketplace payments under Section 194-O of Indian IT Act) and GST on platform commission are legal requirements — not optional features. Settlement-service cannot be used in production without them.

---

## 7. Search, Discovery, and Personalization

| Feature | ZapMarket | Amazon India |
|---|---|---|
| Full-text search (Typesense) | ✅ Client, adapter, sync worker, handler | ✅ Elasticsearch + ML |
| Faceted filtering | ✅ `FilterDrawer.tsx` + backend | ✅ |
| Sort options | ✅ `SortSelect.tsx` + backend | ✅ |
| Search autocomplete | ✅ `SearchModal.tsx` | ✅ |
| Category navigation | ✅ `CategoryGrid.tsx` + backend | ✅ |
| Product recommendations | ❌ | ✅ Collaborative filtering |
| Recently viewed | ❌ | ✅ |
| Trending / curated sections | ❌ UI components exist, no backend data | ✅ |
| Search result ranking (by sales, rating, seller tier) | ❌ No boosting in Typesense adapter | ✅ |
| Personalization layer | ❌ Zero user event tracking | ✅ |
| Flash sale / countdown (backend-driven) | ❌ CountdownTimer UI but no backend data | ✅ |

---

## 8. Reviews and Returns

| Feature | ZapMarket | Amazon India |
|---|---|---|
| Review moderation (PENDING_MODERATION) | ✅ | ✅ |
| Review with images | ✅ `image_urls TEXT[]` | ✅ |
| Verified purchase check | ✅ `verified_purchase` column + use case guard | ✅ |
| Rating aggregation on product | ❌ No materialized view or aggregate table | ✅ |
| Return request (reason + description) | ✅ | ✅ |
| Multi-item return in one request | ❌ One `order_item_id` per return row | ✅ |
| Return images / evidence | ❌ No `image_urls` on returns | ✅ |
| Return approval workflow | ❌ Status column exists, no transitions wired | ✅ |
| Return pickup scheduling | ❌ | ✅ |
| Exchange (not just return) | ❌ | ✅ |
| Return debit to seller | ❌ | ✅ |

---

## 9. Security

| Area | ZapMarket | Amazon India |
|---|---|---|
| JWT (RS256) + refresh tokens | ✅ Redis blacklist on logout | ✅ |
| Social OAuth (Google + Facebook) | ✅ | ✅ + Apple |
| OTP verification (phone + email) | ✅ Hashed, expiry enforced | ✅ |
| MFA at login (TOTP/HOTP) | ❌ | ✅ |
| Rate limiting on login/register | ❌ Only OTP routes protected | ✅ |
| Account lockout after failed attempts | ❌ No attempt counter | ✅ |
| Device/session tracking | ❌ No device fingerprint on tokens | ✅ |
| Secrets management | ❌ Env vars in compose | ✅ Secrets Manager |
| Stripe webhook signature validation | ✅ | ✅ |
| GSTIN/PAN stored for sellers | ✅ | ✅ |
| Data at rest encryption | ❌ Not configured | ✅ |
| SQL injection surface | Low (raw sql.DB, parameterized) | ✅ |

---

## 10. Reliability and Ops

| Area | ZapMarket | Amazon India |
|---|---|---|
| Metrics (Prometheus) | ✅ All services scraped | ✅ |
| Dashboards (Grafana) | ✅ 1 dashboard | ✅ Dozens + auto-alerting |
| Log aggregation (Loki + Promtail) | ✅ | ✅ CloudWatch |
| Alerting rules | ❌ None defined | ✅ |
| Distributed tracing | ❌ | ✅ X-Ray |
| CI/CD pipelines | ❌ None | ✅ |
| Container orchestration | Docker Compose only | ✅ EKS |
| PgBouncer connection pooling | ✅ | ✅ RDS Proxy |
| Health checks across all services | ❌ Partial | ✅ |
| Secrets management | ❌ | ✅ |

---

## Frontend Gaps

| Area | ZapMarket | Amazon India |
|---|---|---|
| buyer-ui component library | ✅ shadcn/ui + Radix + Tailwind v4 | ✅ |
| seller-ui component library | ❌ Tailwind + lucide only | ✅ Full design system |
| Shared UI package across apps | ❌ Each app is fully isolated | ✅ |
| Form validation | ❌ No react-hook-form/zod in any app | ✅ |
| Data fetching/caching | ❌ No react-query or SWR | ✅ |
| Testing | buyer-ui: none; seller-ui: Playwright e2e only | ✅ |
| Accessibility tooling | ❌ | ✅ |
| i18n | ❌ | ✅ (English + Hindi + regional) |
| Next.js version consistency | ❌ buyer-ui=15, others=16 | ✅ |

---

## Prioritized Roadmap

| Change | Effort | Impact | Rationale |
|---|---|---|---|
| Activate Kafka + Debezium (Stage 7) | M | H | Load-bearing: all saga flows are inert without this |
| Add inventory saga compensation on `payment.failed` | S | H | Prevents permanent inventory lock on failed payment |
| Wire promotions-service into order creation | S | H | Coupon system is built; just needs to be called |
| Add `delivery_agents` table + assignment to shipments | S | H | Prerequisite for any last-mile operations |
| Add COD as payment method (enum + reconciliation table) | M | H | ~60% of India volume; blocking market entry |
| Add proof of delivery (OTP + photo_url on shipments) | S | H | Required for COD reconciliation and fraud prevention |
| Wire one real 3PL adapter (Shiprocket REST API) | M | H | CarrierClient interface exists; just needs HTTP impl |
| Add TDS + GST on commission in settlement-service | M | H | Legal requirement under Section 194-O, CGST Act |
| Add seller bank/UPI details + payout flow | M | H | Sellers cannot be paid without this |
| Add seller KYC approval status + admin endpoint | S | M | Compliance and fraud prevention |
| Add `failed_login_attempts` + lockout + rate-limit `/auth/login` | S | H | IPRateLimit middleware already exists; one-line wiring |
| Add `delivery_address` columns to `orders` table | S | H | Shipment cannot be created without a destination |
| Add reattempt logic (`attempt_count`, `next_attempt_at`) on shipments | S | M | Required for standard 3-attempt delivery SLA |
| Add return approval state machine + pickup scheduling | M | M | Return table exists; transitions not wired |
| Add rating aggregation materialized view on products | S | M | Ad-hoc rating query won't scale past 10k reviews |
| Add `reservation_expiry_sweeper` scheduled job | S | M | Expired reservations inflate available stock |
| Implement Razorpay gateway (schema already declares it) | M | H | Primary India gateway; Stripe has lower India acceptance |
| Add Prometheus alerting rules | S | M | Dashboard exists; zero alerts on it |
| Create GitHub Actions CI (`go test ./...` per service) | S | M | Zero tests run on PR today |
| Add react-hook-form + zod to all UIs | M | M | Forms in seller/admin UI have no validation |
| Add react-query for data fetching across UIs | M | M | No caching layer; every nav causes a full refetch |
| Add search result boosting (sales rank, rating) in Typesense adapter | S | M | Relevance degrades without boosting signals |
| Add exchange flow to review-return-service | M | L | Returns handle refund; exchange needs separate path |
| Produce k8s/Helm manifests | L | H | Required for cloud deployment |
| Add MFA (TOTP) at login | M | M | High-value accounts (sellers, admins) need it first |
| Add personalization / recently viewed tracking | L | M | No user event store exists; requires new infrastructure |
