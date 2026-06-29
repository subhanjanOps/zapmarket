# ZapMarket — Architecture Gaps & Missing Features

**Reviewed:** 2026-06-29  
**Reviewer:** Principal Software Architect  
**Baseline:** Amazon, Flipkart, Meesho (production e-commerce at scale)

---

## Critical Gaps

### G1. Synchronous Checkout Saga — No Compensation Logic

**Location:** `services/order-management-service/internal/service/order_service.go`

The checkout flow calls three downstream services over gRPC in sequence:

```
1. inventory-service → ReserveStock
2. payment-service   → ChargeCard
3. inventory-service → DeductStock
4. Write order + outbox to DB
```

**Problems:**
- If step 2 succeeds and step 3 times out, payment is captured but stock is never deducted — silent financial inconsistency
- If the pod crashes between steps 2 and 3, money is taken but no order is confirmed
- No compensating actions (refund, release reservation) are triggered on partial failure
- No retry policy with backoff on any gRPC call

**Required fix:** Choreography-based saga via Kafka events, or a durable workflow engine (Temporal). Each step must publish a domain event; the next step is triggered by consuming that event. Compensating events must be defined for every failure path.

---

### G2. No Search Service

**Missing services:** Elasticsearch / Typesense / OpenSearch  
**Missing workers:** catalog-sync-worker (product → search index)

Buyers can only browse by category (22 flat categories). There is no:
- Keyword search
- Faceted filtering (price range, brand, rating, attributes)
- Relevance ranking
- Autocomplete / search suggestions
- Zero-result tracking or search analytics

The `docker-compose.yml` has a commented-out Elasticsearch config — it is not wired.

**Required fix:**
1. Deploy Typesense (simpler operational model at this scale)
2. Add a `catalog-sync-worker` that consumes `product.created` and `product.updated` Kafka events and indexes into Typesense
3. Replace category-browse endpoints with Typesense faceted search API

---

### G3. No Cart Domain

**Missing service:** `cart-service`

There is no Cart entity anywhere in the codebase. Buyers go directly from product listing to checkout. Real shopping requires:
- Guest cart (session-based, no authentication required)
- Authenticated cart (persisted across devices, synced on login)
- Guest-to-authenticated cart merge on login
- Price re-validation at checkout (SKU prices can change after add-to-cart)
- Quantity cap enforcement per SKU

**Required fix:** New `cart-service` with Redis for guest carts (TTL-based) and PostgreSQL for authenticated carts. Must consume `product.updated` events to invalidate stale prices.

---

## High Severity Gaps

### G4. No Seller Payout / Settlement Service

**Missing service:** `settlement-service` or `seller-ledger-service`

`order_items.seller_id` is captured at checkout but nothing consumes it for financial settlement. Missing:
- Seller ledger (credit on sale, debit on refund, platform commission deduction)
- Payout scheduling (weekly/daily bank transfer)
- TDS deduction (mandatory for Indian marketplaces above ₹30,000/quarter threshold)
- GST-compliant tax invoicing per transaction

**Required fix:** New service consuming `payment.captured` and `payment.refunded` Kafka events. Maintain a double-entry ledger per seller. Integrate Razorpay Payout API for bank transfers.

---

### G5. Single Warehouse Hardcoded in Inventory Service

**Location:** `services/inventory-service/internal/domain/models.go` — `DefaultWarehouseID` constant

The schema supports multiple warehouses (`warehouses` table exists with city, state, pincode columns) but the application ignores it entirely. Issues:
- Cannot model a seller with multiple storage locations
- No pincode-to-warehouse routing at checkout
- No split-order logic for items fulfilling from different warehouses
- Transit inventory not tracked

**Required fix:** Remove `DefaultWarehouseID`. Add a warehouse selection use case that maps buyer delivery pincode to nearest active warehouse using a pincode lookup table.

---

### G6. Reservation TTL Has No Expiry Worker

**Location:** `services/inventory-service/migrations/0001_init.up.sql`

Reservations have an `expires_at` column (15-minute TTL) but there is no background worker that:
- Queries for expired reservations
- Calls `ReleaseStock` to return `qty_reserved` to `qty_available`
- Publishes `inventory.reservation_expired` event so order-management can cancel the order

Without this, expired reservations permanently lock stock.

**Required fix:** Add a ticker-based goroutine in `inventory-service` that polls `reservations WHERE status = 'RESERVED' AND expires_at < NOW()` every 60 seconds and runs the release flow.

---

### G7. No Contract Tests for gRPC or Kafka Schemas

**Missing:** Pact tests, Buf breaking-change checks, Confluent Schema Registry

The system has 7 services communicating over gRPC and Kafka. Any breaking change to a proto file or Kafka event payload will fail silently at runtime. Current risks:
- Proto files are manually copied into each consuming service under `proto/<name>pb/` — there is no generated-output discipline
- Kafka event payloads are raw JSON with no schema enforcement
- A field rename in `payment.proto` will not be caught until the order-management pod crashes

**Required fix:**
1. Adopt `buf.build` workspace at repo root; generate proto output via `buf generate`; remove manual copies
2. Register all Kafka event schemas in Confluent Schema Registry; enable `BACKWARD` compatibility checks
3. Add Pact consumer-driven contract tests for every gRPC client/server pair

---

### G8. API Gateway Has No Circuit Breaker or Retry Policy

**Location:** `services/api-gateway/internal/proxy/`

The custom reverse proxy forwards requests to downstream services with no:
- Circuit breaker (if product-catalog-service is slow, all proxied requests queue up and OOM the gateway)
- Retry with backoff on 5xx responses
- Timeout enforcement per upstream route
- Health-based upstream removal

**Required fix:** Wrap each proxy target with `sony/gobreaker` or `failsafe-go`. Add per-route timeout config (read from the routes DB table). Add upstream health polling that removes unhealthy targets from the routing table.

---

### G9. Inconsistent Internal Architecture Across Services

**Compliant:** `currency-service` (layered: `domain / application / infrastructure / interfaces`)  
**Non-compliant:** `auth-service`, `product-catalog-service`, `order-management-service`, `inventory-service`, `payment-service` (flat: `domain / service / repository / handler`)

The `docs/architecture-principles.md` mandates the layered structure. Five of six backend services do not follow it. This means:
- A developer moving between services faces two different mental models
- Use case isolation cannot be enforced (business logic leaks into handlers and repositories)
- The principles document is aspirational, not actual

**Required fix:** Migrate one service per sprint using `currency-service` as the reference implementation. Recommended order: inventory → payment → order-management → product-catalog → auth.

---

### G10. No Reviews, Ratings, or Returns Domain

**Missing services:** `review-service`, `returns-service`

These are not optional for a marketplace:

| Feature | Status | Business Impact |
|---|---|---|
| Product reviews and star ratings | ❌ Absent | Primary buyer trust signal |
| Verified purchase badge | ❌ Absent | Prevents fake reviews |
| Return / RMA request flow | ❌ Absent | Required by Consumer Protection Act |
| Return-triggered refund automation | ❌ Absent | Manual process currently |
| Seller response to reviews | ❌ Absent | Seller engagement signal |

---

### G11. No Promotions or Pricing Engine

**Missing service:** `promotions-service` or `pricing-service`

There is no concept of:
- Discount codes / coupons
- Flash sales with time-bounded pricing
- Bulk pricing tiers (buy 3, get 10% off)
- Seller-funded vs platform-funded promotions
- Loyalty points / cashback

SKU prices in `product-catalog-service` are static `price_amount` and `compare_price` columns. There is no runtime price computation.

---

### G12. No Shipping / Logistics Integration

**Missing service:** `logistics-service` or carrier integration in order-management

Orders are confirmed but never shipped. Missing:
- Carrier rate shopping (Shiprocket, Delhivery, Bluedart)
- Shipment label generation
- Tracking number capture and propagation
- Delivery status webhooks (out-for-delivery, delivered, failed-delivery)
- Buyer-facing tracking page

---

## Medium Severity Gaps

### G13. Push and SMS Notifications Not Wired

**Location:** `services/notification-service/`

The Twilio SMS adapter exists in `auth-service/sms/` but is never called from `notification-service`. Push notifications (FCM/APNs) have no implementation at all.

Currently only email is sent for order/payment/inventory events. SMS is the primary notification channel for Indian buyers (Flipkart sends SMS for every order state change).

**Required fix:** Wire Twilio into `notification-service`'s notifier interface. Add an FCM publisher for push notifications (requires a device token registry — new table in auth-service).

---

### G14. No Seller Bulk Operations

**Partially planned:** `import-service` (async CSV import, in roadmap per memory notes)

Sellers cannot:
- Bulk upload product catalog via CSV/Excel
- Bulk update prices or stock levels
- Export their order history
- Import from existing platforms (Amazon seller export format)

A seller with 500 SKUs cannot onboard manually via the UI.

---

### G15. No Buyer Wishlist / Saved Items

**Missing:** No wishlist entity or service

Wishlists are a core retention mechanism and conversion signal. They also feed:
- Back-in-stock notifications
- Price-drop notifications
- Abandonment recovery campaigns

---

### G16. No Admin Analytics or Reporting

**Location:** `services/admin-ui/` — stub UI only

The admin panel has route stubs but no data. Missing:
- Platform GMV dashboard (daily/weekly/monthly)
- Order funnel (cart → checkout → confirmed → shipped)
- Inventory health (low stock alerts, dead stock)
- Seller performance metrics
- Fraud signals (unusual order volumes, return rate spikes)

---

### G17. No Load Testing Baseline

No `k6`, `locust`, or `vegeta` scripts exist. The performance profile of the checkout flow under concurrent load is unknown. Critical unknowns:
- How many concurrent reservations can the Redis Lua script sustain?
- What is the p99 latency of the synchronous gRPC chain at checkout?
- At what RPS does the API gateway become the bottleneck?

---

### G18. No Database High Availability

All 8 PostgreSQL databases run as single-instance containers. There is no:
- Read replica for analytics queries
- Streaming replication / failover
- Connection pooling via PgBouncer (each service opens its own pool directly)

A single PostgreSQL crash takes down every service whose DB lives on that host.

---

### G19. Secrets Management via Environment Variables Only

All credentials (JWT secret, DB passwords, Stripe API key, Twilio token) are passed as env vars. There is no:
- HashiCorp Vault integration
- AWS Secrets Manager / GCP Secret Manager rotation
- Secret rotation without pod restart
- Audit trail for secret access

---

### G20. No Mobile Application

The platform is web-only (4 Next.js UIs). In India, 85%+ of e-commerce traffic is mobile-native (Android). Flipkart and Meesho are mobile-first platforms. No React Native or Flutter app exists.

---

## Summary Table

| ID | Gap | Severity | Estimated Effort |
|---|---|---|---|
| G1 | Synchronous checkout saga, no compensation | Critical | 2 weeks |
| G2 | No search service | Critical | 2–3 weeks |
| G3 | No Cart domain | Critical | 1 week |
| G4 | No seller payout / settlement | High | 3 weeks |
| G5 | Single warehouse hardcoded | High | 1 week |
| G6 | Reservation TTL has no expiry worker | High | 2 days |
| G7 | No contract tests (gRPC + Kafka schemas) | High | 1 week setup |
| G8 | API gateway: no circuit breaker or retry | High | 3 days |
| G9 | Inconsistent architecture across services | High | 3–4 weeks per service |
| G10 | No reviews, ratings, or returns domain | High | 3 weeks |
| G11 | No promotions / pricing engine | High | 2 weeks |
| G12 | No shipping / logistics integration | High | 2 weeks |
| G13 | Push and SMS notifications not wired | Medium | 2 days |
| G14 | No seller bulk operations | Medium | 2 weeks |
| G15 | No buyer wishlist | Medium | 3 days |
| G16 | No admin analytics or reporting | Medium | 2 weeks |
| G17 | No load testing baseline | Medium | 1 week |
| G18 | No database high availability | Medium | 1 week |
| G19 | Secrets via env vars only | Medium | 1 week |
| G20 | No mobile application | Medium | 8–12 weeks |

---

## Recommended Fix Order

**Immediate (before any new features):**
1. G6 — Reservation expiry worker (2 days, prevents permanent stock lockup today)
2. G1 — Saga compensation (2 weeks, prevents financial corruption)
3. G8 — API gateway circuit breaker (3 days, prevents cascading failures)

**Next sprint:**
4. G3 — Cart service (unblocks real buyer flows)
5. G7 — Contract tests (protects all subsequent refactors)
6. G2 — Search service (unblocks catalog discoverability)

**Following quarter:**
7. G9 — Architecture migration (inventory → payment → order → catalog → auth)
8. G4 — Seller settlement (required before onboarding real sellers)
9. G5 — Multi-warehouse (required before second fulfillment location)
10. G10 — Reviews and returns (required before public launch)
