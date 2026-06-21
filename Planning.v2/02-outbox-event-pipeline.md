# Phase 2 — Outbox / Event Pipeline

**Goal:** Make the transactional outbox pattern actually work end-to-end. Currently, five services write to `outbox` tables in DB transactions, but no process reads those rows and publishes them to Kafka. The notification-service Kafka consumer is wired but has nothing to consume.

**Current state:**
- Kafka is running in `docker-compose.yml` (healthy)
- `notification-service` has a full Kafka consumer with `pkg/kafka`
- `order-management-service`, `inventory-service`, `payment-service` all write to `outbox` tables
- **Debezium CDC is commented out** in `docker-compose.yml`
- No outbox relay process exists

---

## 2.1 Decision: Debezium CDC vs. Polling Relay

### Option A — Enable Debezium CDC (preferred for production)

Debezium watches PostgreSQL WAL and publishes each new outbox row to Kafka automatically, with exactly-once delivery guarantees.

**Pros:** True CDC, no polling overhead, sub-second latency, no missed rows even under load.  
**Cons:** Requires PostgreSQL `wal_level = logical`, adds a JVM process, more ops surface.

**Steps:**
1. Set `wal_level = logical` in PostgreSQL config (add `command: ["postgres", "-c", "wal_level=logical"]` to the postgres service in `docker-compose.yml`).
2. Uncomment and configure the `kafka-connect` service in `docker-compose.yml`.
3. Write a Debezium connector config JSON for each outbox table:
   ```json
   {
     "name": "order-outbox",
     "config": {
       "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
       "database.hostname": "zapmarket-postgres",
       "database.dbname": "ordermgmt",
       "table.include.list": "public.outbox",
       "transforms": "outbox",
       "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
       "transforms.outbox.table.field.event.type": "event_type",
       "transforms.outbox.table.field.event.id": "aggregate_id"
     }
   }
   ```
4. Register connectors via `kafka-connect` REST API on first boot (use a one-shot init container like `minio-init`).

### Option B — Polling Relay Worker (simpler, implement first)

A `pkg/outbox` relay package polls each `outbox` table every second, publishes to Kafka, and marks rows as processed.

**Pros:** No extra infrastructure, works with existing PostgreSQL config.  
**Cons:** 1-second latency floor, polling adds DB load.

**Recommendation:** Implement Option B first (gets events flowing immediately), then migrate to Option A when moving toward production.

---

## 2.2 Polling Relay — Implementation

### `pkg/outbox/relay.go`

```go
type Relay struct {
    db       *sql.DB
    producer kafka.Producer
    table    string  // "outbox" (each service has its own DB)
    logger   *slog.Logger
}

func (r *Relay) Run(ctx context.Context) {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C: r.flush(ctx)
        }
    }
}

func (r *Relay) flush(ctx context.Context) {
    // SELECT ... FOR UPDATE SKIP LOCKED LIMIT 100 ORDER BY id
    // Publish each row to Kafka topic = event_type (e.g. "order.created")
    // UPDATE outbox SET processed_at = NOW() WHERE id = ANY($1)
}
```

Key design points:
- `FOR UPDATE SKIP LOCKED` allows multiple relay instances without duplicate delivery.
- Batch size 100 per tick.
- Kafka publish failure → do NOT mark as processed → will retry next tick.
- Add `processed_at TIMESTAMPTZ` column to all outbox tables (migration needed).

### Schema migration

Add to each service's migrations:
```sql
ALTER TABLE outbox ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_outbox_unprocessed ON outbox (id) WHERE processed_at IS NULL;
```

### Wire into each service's `cmd/main.go`

```go
relay := outbox.NewRelay(db, kafkaProducer, "outbox", logger)
go relay.Run(ctx)
```

---

## 2.3 Kafka Topic Design

| Topic | Producer | Consumer |
|-------|----------|----------|
| `order.created` | order-management-service | notification-service |
| `order.confirmed` | order-management-service | notification-service |
| `order.cancelled` | order-management-service | notification-service |
| `payment.captured` | payment-service | notification-service |
| `payment.failed` | payment-service | notification-service |
| `payment.refunded` | payment-service | notification-service |
| `inventory.reserved` | inventory-service | (future: analytics) |
| `inventory.released` | inventory-service | (future: analytics) |
| `inventory.depleted` | inventory-service | notification-service |

Topic config: `replication-factor=1` (single-node dev), `partitions=3`, retention 7 days.

Create topics on first boot via a `kafka-init` one-shot container (similar to `minio-init`).

---

## 2.4 Dead Letter Queue

Add a `DLQ` topic (`zapmarket.dlq`) and a DLQ consumer in notification-service that logs failed messages with their headers for manual replay.

```go
// On max retries exceeded in consumer
producer.Publish(ctx, "zapmarket.dlq", msg)
```

---

## 2.5 Outbox Retention / Cleanup

Processed outbox rows should be pruned to prevent unbounded table growth.

Add a cleanup job to each service:
```sql
DELETE FROM outbox WHERE processed_at < NOW() - INTERVAL '7 days';
```

Run via a goroutine on a daily ticker, or as a PostgreSQL scheduled job.

---

## Acceptance criteria

- `order.confirmed` event reaches `notification-service` within 2 seconds of checkout saga completing
- Duplicate events (same `outbox_id`) are deduplicated by `notification-service`
- Relay survives Kafka being temporarily down (retries, does not crash)
- Outbox rows older than 7 days are pruned

---

## Estimated effort

1 week for polling relay + Kafka topic init. 1 additional week for Debezium migration.
