# Plan 01 — Activate Kafka + Debezium

**Effort:** M | **Impact:** H | **Blocks:** Plans 02, 03, 04, 05, 06

## Context
Every saga step (order→inventory→payment→notification) writes to an `outbox` table but nothing reads it. Debezium is config-referenced but commented out of docker-compose. Until Kafka is live, every order stays permanently `PENDING`.

## Scope
- Uncomment Kafka + Zookeeper + Debezium in `docker-compose.yml`
- Create Debezium connector configs for each service's `outbox` table
- Verify consumer groups in inventory-service, payment-service, notification-service connect and consume
- Smoke test: place an order end-to-end and confirm status transitions from `PENDING` → `CONFIRMED` → `PAYMENT_CAPTURED`

## Out of scope
- Kafka cluster sizing / replication (single-broker dev setup is fine for now)
- Schema registry

## Tasks
- [ ] Uncomment Kafka, Zookeeper, and Debezium services in `docker-compose.yml`
- [ ] Create `debezium/connectors/` directory with one JSON connector config per service outbox (`orders`, `inventory`, `payments`)
- [ ] Verify `KAFKA_BOOTSTRAP_SERVERS` env var is set in each consumer service's `.env.example`
- [ ] Start stack and confirm topics `order.created`, `inventory.reserved`, `payment.captured`, `payment.failed` are created
- [ ] Verify `checkout_consumer.go` (inventory-service) subscribes and processes `checkout.requested`
- [ ] Verify `saga_consumer.go` (order-management-service) processes `payment.captured` and updates order status
- [ ] Verify notification-service consumer fires on `order.confirmed` / `payment.captured`
- [ ] Update `outbox` rows: confirm `published_at` is no longer NULL after Debezium drains them

## Done criteria
- `docker compose up` brings Kafka, Debezium, and all consumers up without error
- End-to-end order goes through all status transitions
- No outbox rows with `published_at = NULL` older than 30 seconds under normal operation
