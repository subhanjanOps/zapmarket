# Stage 7 — Event Bus & Outbox Activation ✅ Complete

Corresponds to checklist **Phase 2 (Kafka portion)** and the remainder of **Phase 12**.

## Goal

Turn on real Kafka infrastructure, wire all services to publish events via the
transactional outbox pattern, and connect Notification as the first real consumer —
converting the system from isolated services to an event-driven distributed system.

## Implementation notes

**Debezium decision**: the original plan called for Debezium CDC. We implemented a
lightweight Go polling relay instead (`internal/relay/outbox_relay.go`). Each service
that writes to `outbox` runs its own relay goroutine that polls every 2 seconds and
publishes unpublished rows to Kafka, then marks them `published_at`. This avoids the
operational complexity of Debezium + Kafka Connect during local development, while
maintaining the same correctness guarantees (outbox write is in the same DB transaction
as the business write; relay provides at-least-once delivery). PostgreSQL `wal_level=logical`
is not required. Debezium remains viable for production if CDC fan-out is needed.

**Kafka image**: `apache/kafka:latest` (KRaft mode, no Zookeeper).

## Completed tasks

### 7.1 Infrastructure
- [x] Kafka (`apache/kafka:latest`) running in KRaft mode in `docker-compose.yml`
- [x] Redis running (`redis:7-alpine`) with healthcheck
- [x] Kafka UI (`provectuslabs/kafka-ui`) at http://localhost:8090
- [x] `KAFKA_AUTO_CREATE_TOPICS_ENABLE=true` — topics are created automatically

### 7.2 `pkg/kafka`
- [x] `Producer` wrapper (`pkg/kafka/producer.go`)
- [x] `Consumer` wrapper with consumer-group support, retry-with-backoff, dead-letter
      logging (`pkg/kafka/consumer.go`)
- [x] `Message` type (`pkg/kafka/message.go`)
- [x] Topic name constants — single source of truth (`pkg/kafka/topics.go`):
      `TopicOrders`, `TopicPayments`, `TopicInventory`

### 7.3 Notification wired to real Kafka
- [x] `notification-service` consumes from `orders` topic via `pkg/kafka` consumer
- [x] Retry loop: 5s backoff, fresh consumer per attempt — service never exits on error
- [x] Redis dedup: `SET notif:dedup:{outbox_id} 1 EX 3600 NX`

### 7.4 Payment outbox
- [x] `outbox` table already in payment migration `0001_init.up.sql`
- [x] `MarkCaptured` writes `payment.processed` event to outbox in same transaction
- [x] `MarkFailed` writes `payment.failed` event to outbox in same transaction
      (MarkFailed upgraded from plain `ExecContext` to `WithTransaction`)
- [x] Outbox relay goroutine in `payment-service/main.go` publishing to `TopicPayments`

### 7.5 Inventory outbox
- [x] Migration `0003_add_outbox.up.sql` adds `outbox` table to inventory DB
- [x] `ReserveStock` writes `inventory.reserved` event to outbox in same transaction
- [x] `ReleaseStock` writes `inventory.released` event to outbox in same transaction
- [x] Outbox relay goroutine in `inventory-service/main.go` publishing to `TopicInventory`

### 7.6 End-to-end verification
- [x] Checkout flow: Order → Kafka → Notification logs `order.created`
- [x] Inventory outbox: `inventory.reserved` row published to Kafka on reservation
- [x] Payment outbox: `payment.processed` row published to Kafka on capture
- [x] All service containers start: `connected to Redis` + `outbox relay started`

## Definition of done — met

- `docker compose up -d` brings up Postgres + Kafka + Redis with no manual steps ✅
- Checkout results in outbox rows in order, inventory, and payment DBs, all published via relay ✅
- Notification consumes `order.created` events and logs them ✅
- Restarting notification-service doesn't lose events (consumer group offset commit) ✅
