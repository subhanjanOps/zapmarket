# Plan 06 — Shiprocket 3PL Adapter

**Effort:** M | **Impact:** H | **Depends on:** Plan 04

## Context
`CarrierClient` interface exists in logistics-service with `CreateShipment(ctx, req) → (awb, trackingURL, error)`. The only implementation is a test stub that returns a hardcoded "Delhivery" AWB. No real HTTP call to any courier API exists.

Shiprocket is the standard first-mile aggregator for India — it routes to Delhivery, BlueDart, Ekart, DTDC automatically based on pincode serviceability.

## Scope
- Implement `ShiprocketCarrierClient` satisfying the existing `CarrierClient` interface
- Wire it into logistics-service via config flag so stub remains usable in tests
- Add tracking webhook endpoint to receive Shiprocket push updates

## Out of scope
- Multi-courier fallback (Shiprocket handles this internally via its auto-allocation)
- Label printing UI
- NDR (Non-Delivery Report) portal

## Tasks
- [ ] Add `SHIPROCKET_EMAIL`, `SHIPROCKET_PASSWORD`, `SHIPROCKET_CHANNEL_ID` to logistics-service `.env.example`
- [ ] Create `internal/carrier/shiprocket/client.go`: authenticate with Shiprocket (`POST /v1/external/auth/login`), cache JWT token with refresh
- [ ] Implement `CreateShipment`: call `POST /v1/external/orders/create/adhoc`, map response AWB + tracking URL to `CarrierResponse`
- [ ] Implement `GetTrackingStatus(awb string)`: call `GET /v1/external/courier/track/awb/{awb}`, map to `TrackingEvent`
- [ ] Add `POST /v1/webhooks/shiprocket` HTTP endpoint in logistics-service: parse Shiprocket push payload, write `tracking_events` row, emit `shipment.tracking_updated` event to outbox
- [ ] In `logistics-service/pkg/config/config.go`: add `CARRIER_PROVIDER` env var (values: `stub`, `shiprocket`); wire correct impl via factory
- [ ] Buyer-facing `GET /v1/shipments/{id}/tracking` endpoint: return `tracking_events` ordered by `occurred_at`

## Done criteria
- `docker compose up` with `CARRIER_PROVIDER=shiprocket` creates a real AWB on order shipment
- Tracking webhook updates `tracking_events` table
- `CARRIER_PROVIDER=stub` still works for local dev and tests
- Token refresh handled transparently (Shiprocket tokens expire in 24h)
