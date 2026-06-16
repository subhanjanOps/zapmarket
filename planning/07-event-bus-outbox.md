# Stage 7 — Event Bus & Outbox Activation

Corresponds to checklist **Phase 2 (Kafka portion)** and the remainder of **Phase 12**.

## Goal

Turn on the real Kafka + Debezium infrastructure that's currently commented out in
`docker-compose.yml`, build `pkg/kafka`, and wire the outbox table (Stage 5) and the
fake consumers (Stage 6) to the real thing — converting every stage built so far from
"works in isolation" to "works as an event-driven system."

## Preconditions
- Stage 5 (Order's outbox table exists, writes go in there).
- Stage 6 (Notification's consumer interface exists, ready to be backed by real Kafka).

## Tasks

### 7.1 Turn on infra
- [ ] Uncomment `zookeeper`, `kafka` blocks in `docker-compose.yml`. Confirm port mappings
      (`9092`, `29092`) don't collide with anything else already running.
- [ ] Add a Debezium connector service to `docker-compose.yml` (currently no config exists
      at all per `db-design.md`/compose — this needs to be created, not just uncommented).
- [ ] Configure Debezium's Postgres connector to watch the `outbox` table in
      `order-management-service`'s DB (and Payment's DB, once it also writes outbox rows —
      see 7.4) and publish to Kafka topics named after `event_type`.
- [ ] Verify Postgres has `wal_level = logical` (required for Debezium CDC) — check
      whether the current Postgres container config in `docker-compose.yml` sets this;
      add it if not.

### 7.2 `pkg/kafka`
- [ ] Producer wrapper: thin wrapper over a Go Kafka client (confirm library choice —
      `segmentio/kafka-go` or `confluentinc/confluent-kafka-go`; recommend `kafka-go` for
      pure-Go, no cgo dependency, simpler Docker builds).
- [ ] Consumer wrapper implementing the `EventConsumer` interface from Stage 6, with
      consumer-group support, retry-with-backoff on handler error, and dead-letter logging
      (full DLQ infra is Stage 12; for now, log-and-skip after N retries is enough).
- [ ] Topic registration/constants in one place (`pkg/kafka/topics.go`) so topic name
      strings aren't duplicated across services.

### 7.3 Wire Notification to real Kafka
- [ ] Swap `FakeEventConsumer` for the real `pkg/kafka` consumer in
      `notification-service`'s `main.go` wiring — the interface from Stage 6 means this
      should be a one-line change plus config.

### 7.4 Outbox publisher path for Payment
- [ ] Add the same `outbox` table + migration to `payment-service` (deferred from Stage 4
      since Kafka wasn't live yet) so `payment.processed`/`payment.failed` flow through
      the same CDC pattern as Order, rather than Payment publishing directly to Kafka
      (`design.md`'s stated rule: outbox over direct publish).

### 7.5 Inventory events
- [ ] Add outbox table to `inventory-service` and emit `inventory.reserved` /
      `inventory.released` rows (deferred from Stage 3).

### 7.6 End-to-end verification
- [ ] Run the full checkout flow from Stage 5 with real Kafka now live; confirm
      Notification actually receives and logs `order.created` via the real bus, not the
      fake one — this is the first true end-to-end integration test of the whole system.

## Out of scope
- Kafka metrics/observability dashboards — Stage 10.
- Multi-broker / production Kafka topology — single-broker dev setup is sufficient here.

## Definition of done
- `docker compose up -d` brings up Postgres + Zookeeper + Kafka + Debezium with no manual
  steps.
- A checkout via Order Management results in a row appearing in `outbox`, which Debezium
  picks up, publishes to Kafka, and Notification consumes and logs — observable via
  `docker compose logs notification-service`.
- Killing and restarting `notification-service` mid-flow doesn't lose events (consumer
  group offset commit behaves correctly).
