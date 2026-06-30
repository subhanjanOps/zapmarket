# Plan 02 — Saga Compensations

**Effort:** S | **Impact:** H | **Depends on:** Plan 01

## Context
When payment fails, inventory reservations are never programmatically released. The `RELEASED` status and `reservation_release` movement type exist in the inventory schema but no saga step writes them. Reservations only expire via `expires_at` TTL — no cleanup job either.

## Scope
- Add a compensation handler in inventory-service that listens to `payment.failed` and releases the reservation
- Add a compensation handler in order-management-service that marks order `CANCELLED` on `payment.failed`

## Out of scope
- Retry logic for the compensation itself (idempotency key covers accidental double-fire)
- Dead letter queue (can be added later)

## Tasks
- [ ] In `inventory-service/internal/consumer/`, add `payment_failed_consumer.go` subscribing to `payment.failed` topic
- [ ] Implement `ReleaseReservation(orderID)` use case: set `inventory_reservations.status = 'RELEASED'`, insert `inventory_movements` row with `movement_type = 'reservation_release'`
- [ ] In `order-management-service/internal/consumer/saga_consumer.go`, add handler for `payment.failed` → set order status to `CANCELLED`
- [ ] Write idempotency guard: if reservation already `RELEASED`, no-op
- [ ] Add `payment.failed` topic to Debezium connector config (from Plan 01)
- [ ] Integration test: mock payment failure, confirm reservation status transitions to `RELEASED`

## Done criteria
- Payment failure → reservation released within one Kafka poll interval
- Order status transitions to `CANCELLED`
- Double-fire of `payment.failed` is a no-op (idempotent)
