# Plan 13 — Reservation Expiry Sweeper

**Effort:** S | **Impact:** M | **Depends on:** Plan 01

## Context
`inventory_reservations` has `expires_at TIMESTAMPTZ` but no job reads it. Expired reservations inflate `qty_reserved` indefinitely, making stock appear unavailable when it isn't. This causes false out-of-stock responses.

## Tasks
- [ ] In inventory-service `main.go`, add a background goroutine on a 5-minute ticker
- [ ] Sweeper query: `UPDATE inventory_reservations SET status = 'RELEASED' WHERE status = 'PENDING' AND expires_at < now() RETURNING id, sku_id, quantity`
- [ ] For each released row: update `inventory_items` — `qty_reserved -= quantity` (within a transaction)
- [ ] Insert `inventory_movements` row with `movement_type = 'reservation_release'` and `reason = 'ttl_expired'`
- [ ] Emit `reservation.expired` event to outbox (saga can use this to cancel orders with unconfirmed reservations)
- [ ] In order-management-service: consume `reservation.expired` → cancel order if still `PENDING`
- [ ] Add metric counter `reservation_sweeper_released_total` (Prometheus)
- [ ] Make ticker interval configurable via `RESERVATION_SWEEP_INTERVAL_SECONDS` env var (default 300)

## Done criteria
- Expired reservations are released within one sweep interval
- `qty_reserved` on `inventory_items` stays accurate
- Orders with expired reservations are cancelled automatically
- Sweeper emits a Prometheus counter observable in Grafana
