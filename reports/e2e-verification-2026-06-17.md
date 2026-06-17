# E2E Verification Report — ZapMarket Branch `features/cluster-setup`

**Date:** 2026-06-17  
**Branch:** `features/cluster-setup` (13 commits ahead of `main`, 143 files changed)  
**Verdict:** PASS *(with findings)*

---

## Scope

All six microservices brought up locally against real infrastructure (PostgreSQL, Redis, MinIO, Kafka). Tests exercised the complete request lifecycle — HTTP handler → service → repository → downstream gRPC → Kafka outbox → notification consumer.

**Services tested:**

| Service | Port | Status |
|---|---|---|
| auth-service | 8080 / gRPC 50051 | ✅ Running |
| product-catalog-service | 8081 / gRPC 50052 | ✅ Running |
| inventory-service | 8082 / gRPC 50053 | ✅ Running |
| payment-service | 8083 / gRPC 50054 | ✅ Running |
| order-management-service | 8084 | ✅ Running |
| notification-service | 8085 | ✅ Running |

**Infrastructure:**

| Component | Image | Status |
|---|---|---|
| PostgreSQL | postgres:16-alpine | ✅ |
| Redis | redis:7-alpine | ✅ |
| MinIO | minio/minio | ✅ |
| Kafka | confluentinc/cp-kafka:7.6.0 (KRaft) | ✅ |

> Note: `bitnami/kafka:3.7` (specified in `docker-compose.yml`) was not available in this Docker environment. `confluentinc/cp-kafka:7.6.0` was used as a drop-in replacement and is fully compatible.

---

## Method

Cold-start of all services locally with `go run`. Infrastructure via `docker compose up`. All requests via `curl` against live HTTP endpoints. All downstream gRPC calls happen internally (order → inventory, order → payment, product-catalog → auth). Kafka pipeline tested end-to-end: checkout → outbox written → relay polls → publishes to Kafka → notification-service consumes → notification dispatched.

---

## Steps

### Phase 1 — Auth Service

1. ✅ **Register buyer** `POST /v1/auth/register` → `201` with access + refresh tokens, `user_id: e56fb8b5`
   ```json
   {"user":{"id":"e56fb8b5-943c-4b9c-9fd3-7cac6bd8dc25","email":"e2e_test@zapmarket.com","role":"buyer","is_verified":true},"access_token":"eyJ...","refresh_token":"eyJ..."}
   ```

2. ✅ **Register seller** `POST /v1/auth/register` → `201`, role `seller`, `user_id: 482d62b8`

3. ✅ **Login** `POST /v1/auth/login` → `200` with new tokens

4. ✅ **Get profile** `GET /v1/auth/me` with valid token → `200` with user object

5. 🔍 **Wrong password** → `{"code":"INVALID_CREDENTIALS","message":"invalid email or password"}` (no information leakage about whether email exists — correct)

6. 🔍 **Missing Authorization header** `GET /v1/auth/me` → `{"error":"missing authorization header","code":401}`

7. 🔍 **Duplicate email** `POST /v1/auth/register` with existing email → `{"code":"USER_ALREADY_EXISTS","message":"user with this email already exists"}`

8. ✅ **Swagger UI** `GET /v1/docs/swagger.json` → `200`, paths: `/login`, `/me`, `/register`, `/refresh`, `/oauth/*`

---

### Phase 2 — Product Catalog Service

9. ✅ **List categories (public)** `GET /api/v1/categories` → `200`, returned 2 seeded categories

10. 🔍 **Create category as seller (admin-only)** → `{"code":"FORBIDDEN","message":"you do not have permission to perform this action"}` — RBAC enforced correctly

11. ✅ **Create product as seller** `POST /api/v1/products` → `201`
    ```json
    {"id":"10b093ea-4af6-4cf5-9a89-577d9af007be","name":"E2E Test Product","status":"ACTIVE"}
    ```

12. ✅ **Get product by ID (public)** `GET /api/v1/products/10b093ea` → `200`

13. ❌ **Create SKU** `POST /api/v1/skus` → `500 INTERNAL_SERVER_ERROR "failed to create sku"` — **Bug: see Findings #1**

14. ✅ **Swagger UI** `GET /v1/docs/swagger.json` → `200`, all 20+ endpoints documented

---

### Phase 3 — Order Management Service (core feature of this branch)

15. ✅ **Full checkout saga** `POST /v1/orders` →
    - Inventory gRPC `ReserveStock` called for SKU `aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee`
    - Payment gRPC `ChargeCard` called → `CAPTURED`
    - Inventory `DeductStock` called
    - Order confirmed atomically with outbox event
    ```json
    {"Status":"CONFIRMED","TotalAmount":19998,"Currency":"INR","PaymentID":"523b1bde-0aa2-41d7-ad92-66d0a0b34caf"}
    ```

16. ✅ **Get order** `GET /v1/orders/{id}` → `200` with full order + items including `ReservationID`

17. ✅ **List orders** `GET /v1/orders` → `200` returns all orders for authenticated user

18. ✅ **Idempotency replay** — same `idempotency_key` sent twice → same `order_id` returned, no duplicate processing

19. ✅ **Cancel confirmed order (FSM enforcement)** `POST /v1/orders/{id}/cancel` on a CONFIRMED order →
    ```json
    {"code":"INVALID_TRANSITION","message":"order is in a terminal state: CONFIRMED"}
    ```

20. 🔍 **Cancel another user's order** (seller tries to cancel buyer's order) →
    ```json
    {"code":"ORDER_NOT_FOUND","message":"order not found"}
    ```
    Correct: ownership check returns 404, not 403 (avoids leaking order existence)

21. 🔍 **Cancel non-existent order** → `{"code":"ORDER_NOT_FOUND","message":"order not found"}`

22. 🔍 **Checkout without auth** → `{"code":"MISSING_TOKEN","message":"authorization header is required"}`

23. 🔍 **Checkout with invalid UUID as idempotency_key** → `{"code":"INVALID_IDEMPOTENCY_KEY","message":"idempotency_key must be a valid UUID"}`

24. ✅ **Swagger JSON** `GET /v1/docs/swagger.json` → `200`, paths present:
    - `/v1/orders [post]` — place order
    - `/v1/orders [get]` — list orders
    - `/v1/orders/{id} [get]` — get order
    - `/v1/orders/{id}/cancel [post]` — cancel order

25. ✅ **Swagger UI** `GET /v1/docs/index.html` → `200`

---

### Phase 4 — Outbox Relay + Kafka Pipeline

26. ✅ **Outbox events written** — all order confirmations and cancellations produce a row in the `outbox` table (same DB transaction as the status change):
    ```
    order.confirmed | 2e6604f8-... | published_at: 2026-06-17 ...
    order.cancelled | bd459fea-... | published_at: 2026-06-17 ...
    ```

27. ✅ **Outbox relay publishes to Kafka** — relay polls every 2s, publishes unpublished rows to the `orders` topic, marks `published_at`. All 6 outbox events processed:
    ```
    SELECT event_type, published_at IS NOT NULL FROM outbox;
    -- All 6 rows: published = t
    ```

28. ✅ **Kafka topic auto-created** — `orders` topic created automatically on first publish (`AllowAutoTopicCreation: true` in producer config)

29. ✅ **Outbox relay graceful Kafka failure** — when Kafka was unavailable, relay logged errors and continued retrying every 2s without crashing or blocking the HTTP server:
    ```
    ERROR outbox relay: failed to publish outbox_id=... error="dial tcp [::1]:29092: connectex: No connection..."
    ```

---

### Phase 5 — Notification Service

30. ✅ **Service starts and connects** — Redis connection, Kafka consumer ready, health endpoint `:8085/health` → `200`

31. ✅ **Consumes from Kafka and dispatches notifications** — after checkout, notification service received and processed all 6 backlogged outbox events:
    ```
    INFO notification dispatched user_id=e56fb8b5 event_type=order.confirmed subject="Your order has been confirmed!" body="Order af71daee-... has been confirmed and payment captured..."
    INFO notification dispatched event_type=order.cancelled subject="Your order has been cancelled" body="Order bd459fea-..."
    INFO notification dispatched user_id=e56fb8b5 event_type=order.confirmed ...
    ```

32. ✅ **New order notification end-to-end** — placed a new order → 5s later notification log shows:
    ```
    INFO notification dispatched user_id=e56fb8b5 event_type=order.confirmed subject="Your order has been confirmed!" body="Order 2e6604f8-... has been confirmed..."
    ```

33. ✅ **Redis dedup keys set** — `KEYS notif:dedup:*` returned 6 keys, one per processed outbox event

34. 🔍 **Kafka unavailable at startup** — when Kafka wasn't reachable at startup, the service connected to Redis, logged startup, then exited with:
    ```
    ERROR consumer exited with error: failed to dial: ... connectex: No connection could be made...
    exit status 1
    ```
    Service exits rather than retrying — **see Findings #2**

---

## Findings

### ⚠️ Bug #1 — SKU creation always returns 500

**Path:** `POST /api/v1/skus`  
**Service:** product-catalog-service  
**Root cause:** `sku.VariantAttrs` is typed `interface{}` in the domain model. The `database/sql` driver cannot convert a Go `map[string]interface{}` to PostgreSQL `JSONB` automatically. The value must be marshalled to `json.RawMessage` before being passed to `ExecContext`.  
**Impact:** No SKU can be created via the API. This is a pre-existing bug (not introduced by this branch's commits).  
**Observed:**
```
POST /api/v1/skus → 500 {"code":"INTERNAL_SERVER_ERROR","message":"failed to create sku"}
```

---

### ⚠️ Bug #2 — Notification service exits on Kafka connection failure

**Path:** `services/notification-service/main.go` → `orderConsumer.Run(ctx, ...)`  
**Root cause:** `kafka.Consumer.Run()` returns an error when the broker connection fails, and `main.go` calls `os.Exit(1)` on that error. There is no reconnect loop.  
**Impact:** In environments where Kafka starts slowly (or restarts), the notification service won't recover automatically. It requires an external restart (Docker `restart: unless-stopped` handles this in production, but the process dies which produces noisy alerts).  
**Observed:**
```
INFO  notification service started, consuming events
ERROR consumer exited with error: failed to dial: ... connectex: No connection could be made...
exit status 1
```

---

### ⚠️ Bug #3 — `order.cancelled` payload missing `user_id` on payment failure

**Path:** `services/order-management-service/internal/service/order_service.go` lines ~133–134 and ~140–143  
**Root cause:** When the checkout saga cancels due to payment failure, `MarkCancelled` is called with a payload of:
```json
{"order_id":"...", "reason":"payment_failed", "payment_status":"FAILED"}
```
No `user_id` is included. The `CancelOrder` method (user-initiated) correctly includes `user_id`. This inconsistency means the notification service logs `user_id=""` for payment-failure cancellations.  
**Observed:**
```
INFO notification dispatched user_id="" event_type=order.cancelled subject="Your order has been cancelled"
```
When a real email provider replaces `LogNotifier`, notifications for payment-failure cancellations will have no recipient.

---

### ℹ️ Observation — `bitnami/kafka:3.7` not available in local Docker

`docker-compose.yml` specifies `bitnami/kafka:3.7` but that tag is unavailable in this Docker environment. `confluentinc/cp-kafka:7.6.0` works as a drop-in with equivalent KRaft configuration. Consider adding the confluentinc image as a fallback or pinning a bitnami tag that's confirmed pullable.

---

### ℹ️ Observation — No `.env` files for inventory-service and payment-service

Both services lack a `.env` file. Without it, they start on the default `HTTP_PORT=8080` / `GRPC_PORT=50051` (same as auth-service) and crash with bind errors. The `.env.example` files exist but aren't copied. Consider adding a `make setup-env` target or noting this in the README.

---

### ℹ️ Observation — Domain struct field names are Go-style (not JSON-style) in responses

Order API responses use Go struct field casing (`"ID"`, `"UserID"`, `"SKUID"`) instead of JSON snake_case (`"id"`, `"user_id"`, `"sku_id"`). This is because the `domain.Order` and `domain.OrderItem` structs have no `json:"..."` tags. All other services use json-tagged structs. API clients consuming the order endpoints will get `ID` not `id`.  
**Example:**
```json
{"ID":"af71daee-...","UserID":"e56fb8b5-...","Status":"CONFIRMED","TotalAmount":19998}
```

---

## Summary

| Category | Result |
|---|---|
| Auth: register / login / profile / token errors | ✅ All pass |
| Auth: RBAC enforcement on catalog admin routes | ✅ Pass |
| Product catalog: read endpoints (categories, products) | ✅ Pass |
| Product catalog: SKU creation | ❌ Fails (pre-existing VariantAttrs bug) |
| Order checkout saga (reserve → pay → confirm) | ✅ Pass |
| Order idempotency | ✅ Pass |
| Order cancel (new endpoint) | ✅ FSM enforced, ownership enforced |
| Order swagger docs (new) | ✅ All 4 routes documented |
| Outbox relay → Kafka publish | ✅ Pass |
| Notification service → Kafka consume + dispatch | ✅ Pass |
| Redis dedup in notification service | ✅ Pass |
| Notification service resilience (Kafka down at start) | ⚠️ Exits, no retry |
| Cancelled order `user_id` in payload | ⚠️ Missing on payment-failure path |
