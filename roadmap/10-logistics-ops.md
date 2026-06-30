# Plan 10 — Logistics Ops (Reattempts, Return Pickup, Reverse Logistics)

**Effort:** M | **Impact:** M | **Depends on:** Plans 04, 06

## Context
After delivery agent layer (Plan 04) and 3PL adapter (Plan 06) are in place, need the operational flows: reattempt scheduling after failed delivery, and return pickup (reverse logistics).

## Scope
- Reattempt scheduling: auto-schedule next attempt after failed delivery
- Return pickup: when a return request is approved (Plan 11), create a reverse shipment
- Expose tracking to buyer

## Tasks

### Reattempt scheduling
- [ ] In `POST /v1/shipments/{id}/attempt` (Plan 04): on attempt failure, if `attempt_count < 3`, set `next_attempt_at = now() + 24h`, status = `REATTEMPT_SCHEDULED`; if `attempt_count >= 3`, status = `UNDELIVERED`, emit `shipment.undelivered` event
- [ ] Add consumer in order-management-service for `shipment.undelivered`: set order status to `DELIVERY_FAILED`, notify buyer
- [ ] Notify buyer after each failed attempt via notification-service

### Reverse logistics (return pickup)
- [ ] Add `shipment_type TEXT DEFAULT 'FORWARD' CHECK IN ('FORWARD', 'REVERSE')` to `shipments`
- [ ] Add `parent_shipment_id UUID REFERENCES shipments(id)` for reverse shipments
- [ ] `POST /v1/return-shipments`: create reverse shipment row, call 3PL adapter `CreateReversePickup` (Shiprocket supports this via `POST /v1/external/orders/create/return`)
- [ ] Wire return approval (Plan 11) to call this endpoint

### Buyer tracking API
- [ ] `GET /v1/orders/{id}/tracking`: proxy to logistics-service, return shipment status + tracking events array
- [ ] Add route to api-gateway migrations

## Done criteria
- 3 failed attempts → order status `DELIVERY_FAILED`, buyer notified
- Return approval triggers reverse pickup creation with Shiprocket
- Buyer can query order tracking via api-gateway
