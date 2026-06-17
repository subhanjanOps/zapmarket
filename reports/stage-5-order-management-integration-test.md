# Integration Test Report — Stage 5: Order Management & Saga Orchestration

**Date:** 2026-06-17  
**Tester:** Claude Code (automated)  
**Verdict: ✅ PASS**

---

## Environment

| Component | Location | Port |
|---|---|---|
| `order-management-service` | Local (`go run .`) | HTTP 8084 |
| `inventory-service` | Docker (`zapmarket-inventory-service`) | HTTP 8082 / gRPC 50053 |
| `payment-service` | Docker (`zapmarket-payment-service`) | HTTP 8083 / gRPC 50054 |
| `auth-service` | Docker (`zapmarket-auth-service`) | HTTP 8080 / gRPC 50051 |
| PostgreSQL | Docker (`zapmarket-postgres`) | 5432 |

**Test SKU:** `aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee` (seeded with 200 units via `inventory.AddStock` gRPC)  
**Test user:** `ordertest@zapmarket.dev` (role: `buyer`)

---

## Bugs Found and Fixed During Testing

### Bug 1 — Status casing migration not applied to live DB (infrastructure)

**Root cause:** The status enum values for `inventory_reservations`, `payments`, and `refunds` were changed from lowercase (`reserved`, `captured`, etc.) to `UPPER_SNAKE_CASE` in the Go source files and migration `0001` files. However, the Docker images had been rebuilt from the updated source *after* the migration was already at version 1 in the live DB — so `golang-migrate` skipped re-running it. The new Docker binaries inserted `'RESERVED'` which violated the still-lowercase `CHECK` constraint in the DB.

**Symptom:** Every `POST /v1/orders` returned `500 INVENTORY_ERROR: failed to reserve stock`. Confirmed by running the INSERT directly against Postgres: `ERROR: new row for relation "inventory_reservations" violates check constraint`.

**Fix:** Added migration `0002` to inventory-service and payment-service, and `0003` to product-catalog-service, each of which:
- Drops the old lowercase `CHECK` constraint
- Updates any existing rows to UPPER_SNAKE_CASE
- Adds the new UPPER_SNAKE_CASE constraint and DEFAULT
- Recreates affected partial indexes

Files created:
- `services/inventory-service/migrations/0002_status_upper_snake_case.up.sql`
- `services/inventory-service/migrations/0002_status_upper_snake_case.down.sql`
- `services/payment-service/migrations/0002_status_upper_snake_case.up.sql`
- `services/payment-service/migrations/0002_status_upper_snake_case.down.sql`
- `services/product-catalog-service/migrations/0003_status_upper_snake_case.up.sql`
- `services/product-catalog-service/migrations/0003_status_upper_snake_case.down.sql`

---

### Bug 2 — `DeductStock` not called after payment confirmation (logic)

**Root cause:** The checkout saga correctly called `inventory.ReserveStock` (increments `qty_reserved`, creates reservation row) and on payment success called `repo.MarkConfirmed` — but never called `inventory.DeductStock`. `DeductStock` is what transitions the reservation from `RESERVED → CONFIRMED` and permanently decrements `qty_on_hand`. Without it, units were stuck in `qty_reserved` indefinitely and `qty_on_hand` never decreased.

**Symptom:** After a confirmed order for 3 units, stock showed `qty_on_hand=200 qty_reserved=3 qty_available=197` — available was correct (preventing oversell) but `qty_on_hand` should have dropped to 197 to reflect the permanent sale.

**Fix:** Added `DeductStock` method to `internal/clients/inventory_client.go` and wired it into step 5a of the saga in `internal/service/order_service.go`, called per reservation ID after payment capture and before `MarkConfirmed`. Deduct errors are logged but not fatal — payment already captured; stock reconciliation is possible; the order should still confirm.

**Verification:** After fix, a 5-unit confirmed order produced:  
`qty_on_hand: 200 → 195` ✅

---

## Test Scenarios

### S1 — Successful checkout

**Input:**
```json
POST /v1/orders
Authorization: Bearer <buyer-token>

{
  "idempotency_key": "f189317d-ca87-4507-b5dd-946cfdb458bb",
  "currency": "INR",
  "items": [{ "sku_id": "aaaaaaaa-...", "quantity": 3, "unit_price": 50000 }]
}
```

**Response:** `201 Created` in **202ms**
```json
{
  "success": true,
  "data": {
    "ID": "759a8e96-dbf3-4a35-b8c3-9dad36e7abe7",
    "Status": "CONFIRMED",
    "TotalAmount": 150000,
    "Currency": "INR",
    "PaymentID": "3395c33a-f345-4138-96bf-d90a011bd35d",
    "CreatedAt": "2026-06-16T21:49:18.434676Z"
  }
}
```

**DB (orders):** `status=CONFIRMED`, `payment_id` set ✅  
**DB (outbox):** `event_type=order.confirmed` for this order ✅  
**Inventory:** `qty_available` 200 → 197 (3 reserved), `qty_on_hand` unchanged until DeductStock ✅

---

### S2 — Idempotency replay

**Input:** Identical body and `idempotency_key` as S1, sent again.

**Response:** `201 Created`
```json
{ "data": { "ID": "759a8e96-...", "Status": "CONFIRMED" } }
```

**Same `order_id`?** ✅ YES — `759a8e96-dbf3-4a35-b8c3-9dad36e7abe7`  
**Stock unchanged?** ✅ `qty_available=197` (no double-reserve)  
**Orders with this idempotency key in DB:** `1` ✅

---

### S3 — Payment failure (FakeGateway magic decline)

**Input:** `unit_price=10013` paise → total `10013` paise ends in `13`, which `FakePaymentGateway` treats as a card decline.

```json
POST /v1/orders
{ "idempotency_key": "...", "items": [{ "sku_id": "aaaaaaaa-...", "quantity": 1, "unit_price": 10013 }] }
```

**Response:** `409 Conflict` in **200ms**
```json
{
  "success": false,
  "code": "PAYMENT_FAILED",
  "message": "payment was not captured (status: FAILED)"
}
```

**Stock restored?** ✅ `qty_available=197` (unchanged — reservation released via compensation)  
**Order in DB:** `status=CANCELLED` ✅  
**Outbox:** `event_type=order.cancelled` ✅

---

### S4 — Insufficient stock

**Input:** `quantity=9999` against 197 available units.

**Response:** `409 Conflict`
```json
{
  "success": false,
  "code": "INSUFFICIENT_STOCK",
  "message": "insufficient stock for sku aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
}
```

Stock unchanged ✅ No order created in DB ✅

---

### S5 — List orders (GET /v1/orders)

**Response:** `200 OK` — returned all orders belonging to the authenticated buyer. ✅

---

### S6 — Get single order with items (GET /v1/orders/{id})

**Response:** `200 OK`
```json
{ "data": { "order": { "Status": "CONFIRMED" }, "items": [ ... ] } }
```
`status=CONFIRMED`, `items=1` ✅

---

### S7 — Ownership enforcement (different user)

A second buyer (`ordertest2@zapmarket.dev`) attempts to read S1's order.

**Response:** `404 Not Found`
```json
{ "success": false, "code": "ORDER_NOT_FOUND", "message": "order not found" }
```

Returns `404` rather than `403` to prevent order ID enumeration. ✅

---

### S8 — Unauthenticated request (no token)

**Response:** `401 Unauthorized`
```json
{ "success": false, "code": "MISSING_TOKEN", "message": "authorization header is required" }
```
✅

---

### S9 — Invalid idempotency key format

**Input:** `"idempotency_key": "not-a-uuid"`

**Response:** `400 Bad Request`
```json
{ "success": false, "code": "INVALID_IDEMPOTENCY_KEY", "message": "idempotency_key must be a valid UUID" }
```
✅

---

## Final DB State

```
orders table:
  759a8e96  | CONFIRMED | 150000 | payment_id=3395c33a  ← S1 ✅
  bd459fea  | CANCELLED |  10013 | payment_id=null      ← S3 ✅
  [3 stale PENDING rows from pre-fix runs — see notes]

outbox table:
  order.confirmed → 759a8e96   ← S1
  order.cancelled → bd459fea   ← S3
```

---

## Final Stock State (post DeductStock fix)

Test: 5-unit order confirmed.

| Metric | Before | After | Expected |
|---|---|---|---|
| `qty_on_hand` | 200 | 195 | 195 ✅ |
| `qty_reserved` | 3 | 3 | 3 (prior order, unaffected) ✅ |
| `qty_available` | 197 | 192 | 192 ✅ |

---

## Notes

- **3 stale PENDING orders** exist in the DB from the pre-fix test runs (before migration 0002 applied). They will never transition since there is no reaper/reconciliation job. These should be handled by the TTL sweep job already flagged in `planning/12-production-readiness.md`.
- `qty_reserved=3` from the S1 order remains after the DeductStock fix verification because that order was created before the fix. All new orders correctly reach `qty_reserved=0` after deduction.
- The `FakePaymentGateway` decline trigger (amount ending in `13` paise) is a useful test hook — remains documented in `planning/04-payment-service.md`.
