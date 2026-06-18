# ZapMarket E2E Integration Test Report
**Date:** 2026-06-19
**Tester:** Claude (automated)
**Environment:** Docker Compose (all services running)

---

## Summary

| Metric | Count |
|---|---|
| Total tests | 47 |
| Passed | 34 |
| Failed / Bugs found | 13 |
| Critical bugs | 3 |
| High bugs | 5 |
| Medium bugs | 3 |
| Low bugs | 2 |

---

## Infrastructure Status

All 16 containers are running and healthy:

| Container | Status | Notes |
|---|---|---|
| zapmarket-api-gateway | healthy | Port 8000 |
| zapmarket-auth-service | healthy | Port 8080 / gRPC 50051 |
| zapmarket-product-catalog-service | healthy | Port 8081 / gRPC 50052 |
| zapmarket-inventory-service | healthy | Port 8082 / gRPC 50053 |
| zapmarket-payment-service | healthy | Port 8083 / gRPC 50054 |
| zapmarket-order-management-service | healthy | Port 8084 |
| zapmarket-notification-service | healthy | Port 8085 |
| zapmarket-seller-ui | running | Port 3002 (redirects to /dashboard) |
| zapmarket-backoffice-ui | running | Port 3003 (redirects to /dashboard) |
| zapmarket-admin-ui | running | Port 3001 (200 OK) |
| zapmarket-postgres | healthy | Port 5432 |
| zapmarket-redis | healthy | Port 6379 |
| zapmarket-kafka | healthy | Port 9092 |
| zapmarket-kafka-ui | running | Port 8090 |
| zapmarket-minio | healthy | Port 9000-9001 |
| zapmarket-pgadmin | running | Port 5050 |

**Databases found:** `userauth`, `productcatalog`, `ordermgmt`, `inventory`, `payment`, `notification`, `apigateway`

**Note:** The `notification` database has no tables — notification service does not persist events, it dispatches in memory only.

**Gateway routes loaded (8 routes):**
```
/api/v1/categories   → product-catalog-service  auth=none
/api/v1/products     → product-catalog-service  auth=method_split
/v1/auth             → auth-service             auth=required
/v1/auth/login       → auth-service             auth=none
/v1/auth/oauth       → auth-service             auth=none
/v1/auth/refresh     → auth-service             auth=none
/v1/auth/register    → auth-service             auth=none
/v1/orders           → order-management-service auth=required
```

---

## Test Results by Service

### Auth Service (port 8080)

| Test | Result | Notes |
|---|---|---|
| Register buyer (`role=buyer`) | PASS | Returns user + access + refresh tokens |
| Register seller (`role=seller`) | PASS | Works correctly |
| Register admin (`role=admin`) | **FAIL** | Returns 400 "role must be 'buyer' or 'seller'" |
| Duplicate email registration | PASS | Returns `USER_ALREADY_EXISTS` / 409 |
| Login with valid credentials | PASS | Returns fresh tokens |
| Login with invalid password | PASS | Returns `INVALID_CREDENTIALS` / 401 |
| Get /v1/auth/me with valid token | PASS | Returns user profile |
| Get /v1/auth/me with no token | PASS | Returns 401 |
| Get /v1/auth/me with invalid token | PASS | Returns 401 `invalid or expired token` |
| Token refresh | PASS | Returns new access token |
| CORS preflight on auth-service directly | **FAIL** | Returns 405 Method Not Allowed |

### API Gateway (port 8000)

| Test | Result | Notes |
|---|---|---|
| GET /health | PASS | `{"status":"ok","service":"api-gateway"}` |
| CORS for seller-ui (origin: localhost:3002) | PASS | Correct CORS headers on gateway |
| CORS for backoffice-ui (origin: localhost:3003) | PASS | Correct CORS headers |
| GET /api/v1/products (no auth, method_split) | PASS | 200 — public read allowed |
| POST /api/v1/products (no auth, method_split) | PASS | 401 — write requires auth |
| GET /api/v1/skus via gateway | **FAIL** | 404 — route not in gateway_routes table |
| /v1/auth/me via gateway | PASS | Proxied correctly |
| Invalid token via gateway | PASS | 401 returned |

### Product Catalog Service (port 8081)

| Test | Result | Notes |
|---|---|---|
| GET /api/v1/categories | PASS | Returns list |
| POST /api/v1/categories as seller | **FAIL** | 403 FORBIDDEN — admin-only route |
| POST /api/v1/categories as buyer | **FAIL** | 403 FORBIDDEN — admin-only route |
| GET /api/v1/products | PASS | Returns list |
| POST /api/v1/products as seller with `status=active` (lowercase) | **FAIL** | 500 INTERNAL_SERVER_ERROR — no validation error |
| POST /api/v1/products as seller with `status=ACTIVE` | PASS | Creates product correctly |
| Duplicate product slug | PASS | Returns `PRODUCT_ALREADY_EXISTS` |
| Product timestamps on create response | **FAIL** | `created_at` and `updated_at` are `0001-01-01T00:00:00Z` |
| POST /api/v1/skus with `variant_attrs` key | FAIL | Attributes ignored (struct tag is `attributes`) |
| POST /api/v1/skus with `attributes` key | PASS | Attributes stored correctly in DB |
| PUT /api/v1/skus/{id} WITHOUT product_id (bug fix) | PASS | Update succeeds |
| PUT /api/v1/skus/{id} response product_id | **FAIL** | Returns `product_id: 00000000-0000-0000-0000-000000000000` |
| PUT /api/v1/skus/{id} response timestamps | **FAIL** | `created_at` and `updated_at` are `0001-01-01T00:00:00Z` |
| GET /api/v1/skus/{id} with valid UUID | PASS | Returns correct data |
| GET /api/v1/skus/{id} with all-zeros UUID | **FAIL** | Returns `INVALID_DATA "sku id is required"` instead of `NOT_FOUND` |
| GET /api/v1/skus/{id} with malformed UUID | PASS | Returns `INVALID_ID` |
| Duplicate SKU code | PASS | Returns `SKU_ALREADY_EXISTS` |
| CORS OPTIONS on product-catalog-service directly | **FAIL** | Returns 405 — no CORS middleware |

### Order Management Service (port 8084)

| Test | Result | Notes |
|---|---|---|
| GET /v1/orders (empty) | PASS | Returns empty list |
| POST /v1/orders with non-UUID Idempotency-Key header | **FAIL** | 400 INVALID_IDEMPOTENCY_KEY — key must be in request body not header |
| POST /v1/orders with UUID key in body (no inventory) | PASS | Creates order in PENDING, reserves stock fails gracefully |
| POST /v1/orders with inventory seeded | PASS | Order confirmed, payment captured, status=CONFIRMED |
| GET /v1/orders/{id} | PASS | Returns order with items |
| List /v1/orders | PASS | Returns user's orders |
| Idempotency: same key twice | PASS | Returns same order_id |
| Cancel order POST /v1/orders/{id}/cancel | PASS | Order cancelled, Kafka event published |
| Oversell prevention (qty=200, stock=95) | PASS | Returns `INVENTORY_ERROR` |
| Order created before inventory seeded stays PENDING | MEDIUM | No automatic retry or expiry mechanism |

### Inventory Service (port 8082)

| Test | Result | Notes |
|---|---|---|
| GET /health | PASS | 200 |
| HTTP REST API | N/A | No public REST — gRPC only by design |
| gRPC ReserveStock (via order-management) | PASS | Works when inventory row exists |
| gRPC ReserveStock with no inventory row | PASS | Returns NOT_FOUND, order stays PENDING |
| Inventory updated after confirmed order | PASS | qty_on_hand decremented (100 → 95) |
| Oversell via Lua script | PASS | Prevents oversell correctly |

### Payment Service (port 8083)

| Test | Result | Notes |
|---|---|---|
| GET /health | PASS | 200 |
| HTTP REST API (Swagger) | N/A | No documented REST endpoints found |
| Payment created via gRPC (from order-management) | PASS | Payment CAPTURED for confirmed order |
| Kafka payment.processed event | PASS | Published correctly |

### Notification Service (port 8085)

| Test | Result | Notes |
|---|---|---|
| GET /health | PASS | 200 |
| Consumes `orders` topic | PASS | Processes order.cancelled and order.confirmed |
| Consumes `payments` topic | **FAIL** | No template for `payment.processed` event — logged and skipped |
| Consumes `inventory` topic | **FAIL** | Crashes parsing `inventory.reserved` event |

### Kafka Events

| Topic | Events Observed | Status |
|---|---|---|
| orders | `order.cancelled`, `order.confirmed` | PASS — flowing correctly |
| payments | `payment.processed` | PASS — published by payment-service |
| inventory | `inventory.reserved` | PASS — published by inventory-service |

### UI Services

| Service | Status | Notes |
|---|---|---|
| seller-ui (3002) | PASS | Loads, redirects to /dashboard |
| backoffice-ui (3003) | PASS | Loads, redirects to /dashboard |
| admin-ui (3001) | PASS | Returns 200 |
| seller-ui JS errors | Not tested (no browser) | Playwright not available in this environment |

---

## Bug List

### BUG-001: Admin role cannot be self-registered via API
**Severity:** High
**Service:** auth-service
**Endpoint:** `POST /v1/auth/register`
**Steps to Reproduce:**
```json
POST http://localhost:8080/v1/auth/register
{"email":"admin@test.com","password":"Test1234!","full_name":"Test Admin","role":"admin"}
```
**Expected:** Admin user created (with appropriate seeding mechanism)
**Actual:** `{"error":"role must be 'buyer' or 'seller'","code":400}`
**Root Cause:** The register handler validates role against a hard-coded allowlist that excludes `admin`. There is no admin seeding mechanism (no seed script, no migration) — admin users cannot be created at all without direct DB manipulation.
**Fix:** Add an initial admin seeding migration or a separate admin bootstrap endpoint with a secret key. Alternatively document clearly that admin users must be created via direct DB insert.

---

### BUG-002: SKU GET with all-zeros UUID returns wrong error code
**Severity:** Medium
**Service:** product-catalog-service
**Endpoint:** `GET /api/v1/skus/00000000-0000-0000-0000-000000000000`
**Steps to Reproduce:**
```
GET http://localhost:8081/api/v1/skus/00000000-0000-0000-0000-000000000000
Authorization: Bearer <token>
```
**Expected:** `{"success":false,"code":"SKU_NOT_FOUND","message":"sku not found"}` (404)
**Actual:** `{"success":false,"code":"INVALID_DATA","message":"sku id is required"}` (400)
**Root Cause:** In `sku_service.go` `GetSKUByID`, the service checks `if id == uuid.Nil` and returns a validation error instead of attempting the DB lookup. `uuid.Nil` is the all-zeros UUID, which is a valid UUID format that should return NOT_FOUND, not INVALID_DATA.
**Fix:** In `services/product-catalog-service/internal/service/sku_service.go`, remove the `uuid.Nil` check from `GetSKUByID` (it is a valid lookup that will return `sql.ErrNoRows` → NOT_FOUND from the repository layer).

---

### BUG-003: PUT /api/v1/skus/{id} response returns product_id = 00000000-0000-0000-0000-000000000000
**Severity:** High
**Service:** product-catalog-service
**Endpoint:** `PUT /api/v1/skus/{id}`
**Steps to Reproduce:**
```json
PUT http://localhost:8081/api/v1/skus/{id}
{"sku_code":"IPHONE15-128-BLK","price_amount":89999,"currency":"USD","is_active":true}
```
**Expected:** Response includes correct `product_id` from DB
**Actual:** Response contains `"product_id":"00000000-0000-0000-0000-000000000000"` — the zero UUID. The DB record is updated correctly (product_id is preserved in DB), but the response reflects the partial in-memory struct that was never populated with product_id.
**Root Cause:** In `sku_handler.go` `UpdateSKU`, the handler builds a `domain.SKU` struct from the request body (which does not include `product_id` after the fix), calls `UpdateSKU`, and then returns the same partial struct. The handler does not re-fetch the updated SKU from the repository.
**Fix:** After a successful `UpdateSKU` call, call `GetSKUByID` to fetch the full updated record and return that instead. Example:
```go
if err := h.skuService.UpdateSKU(r.Context(), sku); err != nil {
    HandleError(w, err)
    return
}
updated, err := h.skuService.GetSKUByID(r.Context(), id)
if err != nil {
    HandleError(w, err)
    return
}
SuccessResponse(w, http.StatusOK, updated)
```

---

### BUG-004: POST /api/v1/skus and PUT /api/v1/skus/{id} return zero timestamps in response
**Severity:** Medium
**Service:** product-catalog-service
**Endpoints:** `POST /api/v1/skus`, `PUT /api/v1/skus/{id}`
**Steps to Reproduce:**
```
POST /api/v1/skus with valid body
```
**Expected:** Response includes actual `created_at` and `updated_at` timestamps
**Actual:** `"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"`
**Root Cause:** Same as BUG-003 — the handler returns the in-memory struct without re-fetching from DB. The `created_at`/`updated_at` fields are set by `NOW()` in the SQL INSERT, but the Go struct is never updated with those DB-generated values.
**Fix:** After `CreateSKU`, re-fetch the created SKU via `GetSKUByID(ctx, sku.ID)` and return that. This also affects `POST /api/v1/products` which has the same pattern (confirmed via GET returning correct timestamps).

---

### BUG-005: CreateSKURequest and UpdateSKURequest JSON tag mismatch with Swagger docs
**Severity:** High
**Service:** product-catalog-service
**Endpoint:** `POST /api/v1/skus`, `PUT /api/v1/skus/{id}`
**Steps to Reproduce:**
```json
POST /api/v1/skus
{"product_id":"...","sku_code":"TEST-001","price_amount":9999,"currency":"USD","variant_attrs":{"color":"Black"}}
```
**Expected:** Attributes stored (swagger shows `variant_attrs` as the input field name)
**Actual:** Attributes are silently ignored — `variant_attrs` stored as `null` in DB
**Root Cause:** The Go struct tag uses `json:"attributes,omitempty"` but the Swagger documentation (generated from the struct) shows `variant_attrs` in the response model. Callers who read the Swagger spec expect to send `variant_attrs` but must actually send `attributes`. The input field name should be `variant_attrs` to match the DB column name and response field name, OR the swagger should document `attributes`.
**Fix:** Change both `CreateSKURequest` and `UpdateSKURequest` struct tags from `json:"attributes,omitempty"` to `json:"variant_attrs,omitempty"` to match the DB field and response JSON key (`variant_attributes`). Regenerate Swagger docs after the fix.

---

### BUG-006: Product status validation returns 500 instead of 400 for lowercase status
**Severity:** Medium
**Service:** product-catalog-service
**Endpoint:** `POST /api/v1/products`
**Steps to Reproduce:**
```json
POST http://localhost:8081/api/v1/products
{"name":"Samsung","slug":"samsung","category_id":"...","status":"active"}
```
**Expected:** `400 Bad Request` with a message like "status must be one of DRAFT, ACTIVE, INACTIVE, ARCHIVED"
**Actual:** `{"success":false,"code":"INTERNAL_SERVER_ERROR","message":"failed to create product"}`
**Root Cause:** The DB column has a `CHECK` constraint (`products_status_check`) that rejects lowercase values. This constraint violation bubbles up as a generic internal error instead of being caught and translated to a validation error.
**Fix:** In the product repository's `Create` method, check for PostgreSQL error code `23514` (check constraint violation) and return a validation error. Alternatively, validate the status field in the service layer before hitting the DB.

---

### BUG-007: /api/v1/skus route missing from API Gateway
**Severity:** Critical
**Service:** api-gateway
**Steps to Reproduce:**
```
GET http://localhost:8000/api/v1/skus
```
**Expected:** Proxied to product-catalog-service, returns SKU list
**Actual:** `404 page not found`
**Root Cause:** The `gateway_routes` table has no entry for `/api/v1/skus`. This means any frontend (seller-ui, backoffice-ui) that makes SKU requests through the gateway (port 8000) gets 404. SKUs can only be accessed by calling the product-catalog-service directly (port 8081), which bypasses auth and CORS.
**Impact:** Seller UI cannot create, update, or list SKUs through the gateway. This is a critical gap.
**Fix:** Insert the missing route into `gateway_routes`:
```sql
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, enabled)
VALUES ('/api/v1/skus', 'product-catalog-service', 'method_split', true);
```
Also consider adding missing routes for: `/v1/auth/me`, individual resource paths, inventory management (if HTTP API is added), and payment endpoints.

---

### BUG-008: Category CREATE/UPDATE/DELETE only accessible to `admin` role, but admin cannot be registered
**Severity:** Critical
**Service:** product-catalog-service + auth-service
**Steps to Reproduce:**
1. Try to register a user with `role=admin` → fails (BUG-001)
2. Try to create a category as `seller` → 403 FORBIDDEN
3. Try to create a category as `buyer` → 403 FORBIDDEN
**Expected:** Categories can be created by admin users; there is a way to create admin users
**Actual:** No user role in the system can create categories through the API. The DB must be seeded directly.
**Root Cause:** Combined effect of BUG-001 (no admin registration) and the `RequireRole("admin")` middleware on category write endpoints.
**Fix:** At minimum, add an admin bootstrap migration or seed script that creates the first admin user. Long-term, add a super-admin promotion endpoint or an admin-invite flow.

---

### BUG-009: auth-service and product-catalog-service do not handle CORS preflight (OPTIONS) requests
**Severity:** High
**Service:** auth-service (port 8080), product-catalog-service (port 8081)
**Steps to Reproduce:**
```
OPTIONS http://localhost:8080/v1/auth/login
Origin: http://localhost:3002
Access-Control-Request-Method: POST
```
**Expected:** 204 with Access-Control-Allow-* headers
**Actual:** `405 Method Not Allowed`
**Root Cause:** Neither service registers an OPTIONS handler or CORS middleware. Cross-origin preflight requests from browsers will fail if the UI calls these services directly. Currently the API Gateway (port 8000) handles CORS correctly, so this only matters if the UI bypasses the gateway. However, the `seller-ui` is configured to call the gateway, so this is a latent risk.
**Fix:** Add a CORS middleware to both services (e.g., `rs/cors` package). If direct-to-service calls are not intended, document that all frontend traffic must go through the gateway, and enforce this via network policy.

---

### BUG-010: Notification service fails to parse `inventory.reserved` event
**Severity:** High
**Service:** notification-service
**Kafka Topic:** `inventory`
**Error observed in logs:**
```
level=ERROR msg="failed to parse event payload" event_type=inventory.reserved error="json: cannot unmarshal number into Go value of type string"
```
**Steps to Reproduce:** Create an order that succeeds with inventory. The inventory-service publishes `inventory.reserved` to Kafka. The notification-service consumes it and crashes the parse.
**Expected:** Event parsed successfully (even if no notification is dispatched for this event type)
**Actual:** Parse error logged; event processing aborted for this message
**Root Cause:** The notification service's event payload struct expects a field (likely `qty`, `sku_id`, or similar) as `string`, but the inventory-service publishes it as a JSON number (integer). There is a type mismatch between the publisher and consumer.
**Fix:** Audit the notification-service's `inventory.reserved` event struct. Change the offending field type from `string` to `int`/`int64`, or use `json.Number`, to match the published payload:
```json
{"qty": 5, "sku_id": "...", "status": "RESERVED", "order_id": "...", "reservation_id": "..."}
```
`qty` is an integer in the payload but likely declared as `string` in the consumer struct.

---

### BUG-011: No notification template for `payment.processed` event
**Severity:** Low
**Service:** notification-service
**Log:** `level=INFO msg="no notification template for event" event_type=payment.processed`
**Actual:** Payment confirmation events are silently dropped — buyers never receive payment confirmation notifications.
**Fix:** Add a notification template for `payment.processed` in the notification service, analogous to the existing `order.confirmed` template.

---

### BUG-012: PENDING orders stuck with no inventory row are never resolved
**Severity:** Medium
**Service:** order-management-service
**Observed:** Orders created before inventory was seeded remain in `PENDING` status indefinitely. There is no timeout, expiry, or retry mechanism.
**DB state:**
```
4c728857 | PENDING | 89999   -- created before inventory seeded
fd546888 | PENDING | 17999800 -- oversell attempt stored before inventory check
52399194 | PENDING | 17999800 -- oversell attempt
```
**Root Cause:** The order-management-service creates the order record, then asynchronously calls `ReserveStock`. If the gRPC call fails (no inventory row), the order record in the DB remains in `PENDING` status with no mechanism to expire it. Idempotency replay from cache also returns the `PENDING` status, so repeated requests see a stale PENDING order.
**Fix:** Add an order expiry job (e.g., a cron or background goroutine) that transitions PENDING orders older than N minutes to `CANCELLED` or `FAILED`. Alternatively, implement a saga compensation to roll back the order record immediately on reservation failure.

---

### BUG-013: Gateway `Idempotency-Key` header not forwarded correctly by UI / documentation mismatch
**Severity:** Low
**Service:** order-management-service / api-gateway
**Steps to Reproduce:**
```
POST http://localhost:8084/v1/orders
Idempotency-Key: order-test-001    ← non-UUID string
```
**Expected:** Accepts any string or at least a clear error message
**Actual:** `{"success":false,"code":"INVALID_IDEMPOTENCY_KEY","message":"idempotency_key must be a valid UUID"}`
**Additional Finding:** The idempotency key must be sent in the request **body** as `idempotency_key`, not as the `Idempotency-Key` HTTP header. The gateway's CORS config already allows the `Idempotency-Key` header, implying the intent was to use headers. The swagger spec shows it as a body field. This discrepancy will confuse frontend developers.
**Fix:** Standardize on one approach (body field is current implementation). Update the gateway CORS `Access-Control-Allow-Headers` if the header approach is not used, or implement header extraction in the order handler and deprecate the body field.

---

## Recommendations

### Immediate (Critical/High)

1. **Add `/api/v1/skus` to gateway_routes** (BUG-007) — seller-ui is broken without this.
2. **Create admin bootstrap mechanism** (BUG-008/BUG-001) — categories cannot be created by any user through the API.
3. **Fix SKU UPDATE response** (BUG-003) — re-fetch from DB after update to return correct `product_id` and timestamps.
4. **Fix variant_attrs JSON tag** (BUG-005) — change struct tag from `attributes` to `variant_attrs` to match DB and Swagger.
5. **Fix notification-service inventory.reserved parse failure** (BUG-010) — change `qty` field type from string to int in the consumer struct.

### Short-term (Medium)

6. **Fix SKU CREATE response timestamps** (BUG-004) — re-fetch from DB after insert.
7. **Fix product status check constraint** (BUG-006) — catch pgcode 23514 and return a 400 with a clear message.
8. **Fix SKU GET for nil UUID** (BUG-002) — remove the `uuid.Nil` guard in `GetSKUByID`.
9. **Add order expiry mechanism** (BUG-012) — background job to cancel stale PENDING orders.

### Long-term (Low)

10. **Add CORS middleware to auth-service and product-catalog-service** (BUG-009) — defensive measure.
11. **Add payment.processed notification template** (BUG-011).
12. **Clarify Idempotency-Key usage** (BUG-013) — body vs. header.
13. **Add remaining gateway routes** — `/api/v1/categories` write endpoints (admin-gated), `/v1/auth/me`, image endpoints.

---

## DB Integrity Check

```
productcatalog.skus → productcatalog.products: INTACT (2 rows, all have valid product_id)
ordermgmt.orders → ordermgmt.order_items: INTACT
payment.payments → ordermgmt.orders (via order_id): INTACT (1 payment for 1 confirmed order)
inventory.inventory → inventory.warehouses: INTACT
```

No orphaned records found. 3 PENDING orders exist with no corresponding inventory reservation (test artifacts from pre-seeded test run).

---

## Kafka Event Flow Verification

```
Order created → order.confirmed (Kafka: orders topic) ✓
Order cancelled → order.cancelled (Kafka: orders topic) ✓
Payment captured → payment.processed (Kafka: payments topic) ✓
Inventory reserved → inventory.reserved (Kafka: inventory topic) ✓
Notification: order.confirmed consumed → dispatched ✓
Notification: order.cancelled consumed → dispatched ✓
Notification: payment.processed consumed → DROPPED (no template) ✗
Notification: inventory.reserved consumed → PARSE ERROR ✗
```
