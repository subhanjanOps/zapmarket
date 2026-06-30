# Plan 04 — Delivery Agent Layer

**Effort:** S→M | **Impact:** H | **Depends on:** Plan 01

## Context
The `logistics-service` has `shipments` and `tracking_events` tables but no delivery agent concept. There is no `delivery_agents` table, no assignment column on shipments, no proof of delivery fields, and no reattempt logic. This is prerequisite for COD (Plan 05) and 3PL (Plan 06).

Also: `orders` table has no delivery address — a shipment cannot be dispatched without a destination. Fix this here as it touches order-management-service migration.

## Scope
- Add `delivery_address` to `orders`
- Add `delivery_agents` table to logistics-service
- Add agent assignment FK + attempt tracking to `shipments`
- Add `proof_of_delivery` table
- Expose CRUD API for agents (admin-only) and assignment endpoint

## Out of scope
- Agent mobile app / portal UI (later)
- Route optimization
- Agent authentication (use existing auth-service; agents are users with `role = delivery_agent`)

## Tasks

### Order address (order-management-service)
- [ ] Migration: add `delivery_name TEXT`, `delivery_phone TEXT`, `delivery_address_line1 TEXT`, `delivery_address_line2 TEXT`, `delivery_city TEXT`, `delivery_state TEXT`, `delivery_pincode TEXT` to `orders`
- [ ] Update order creation request body and handler to accept and store address
- [ ] Pass `delivery_pincode` to inventory-service for zone validation (already has `pincode_zones` table)

### Delivery agents (logistics-service)
- [ ] Migration: create `delivery_agents` table (`id UUID PK`, `user_id UUID`, `name TEXT`, `phone TEXT`, `vehicle_type TEXT`, `zone TEXT`, `status TEXT DEFAULT 'AVAILABLE'`, `created_at TIMESTAMPTZ`)
- [ ] Migration: add `assigned_agent_id UUID REFERENCES delivery_agents(id)`, `attempt_count INT DEFAULT 0`, `next_attempt_at TIMESTAMPTZ`, `last_attempt_at TIMESTAMPTZ` to `shipments`
- [ ] Migration: create `proof_of_delivery` table (`id UUID PK`, `shipment_id UUID FK`, `method TEXT` CHECK IN `('OTP','SIGNATURE','PHOTO')`, `otp_verified BOOL`, `photo_url TEXT`, `delivered_at TIMESTAMPTZ`, `delivered_by UUID FK delivery_agents`)
- [ ] Add `POST /v1/agents` (admin), `GET /v1/agents`, `PUT /v1/shipments/{id}/assign` endpoints
- [ ] Add `POST /v1/shipments/{id}/attempt` endpoint: increments `attempt_count`, sets `last_attempt_at`, sets `next_attempt_at = now() + 24h` on failure
- [ ] Add `POST /v1/shipments/{id}/deliver` endpoint: writes `proof_of_delivery` row, sets `shipment.status = DELIVERED`
- [ ] Emit `shipment.delivered` event to outbox (needed by payment-service for COD in Plan 05)

## Done criteria
- Shipment can be assigned to a delivery agent via API
- Delivery attempt recorded with timestamp and outcome
- Proof of delivery (OTP or photo) persisted
- `shipment.delivered` event appears in outbox
- Order creation accepts and stores delivery address
