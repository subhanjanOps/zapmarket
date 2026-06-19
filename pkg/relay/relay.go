// Package relay polls the transactional outbox table and publishes unpublished
// events to Kafka. It acts as a lightweight replacement for Debezium during
// local development. Each published row is marked with published_at so it is
// never delivered twice.
package relay

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

type outboxRow struct {
	ID            uuid.UUID
	AggregateID   uuid.UUID
	AggregateType string
	EventType     string
	Payload       json.RawMessage
}

// OutboxRelay polls the outbox table and publishes events to Kafka.
type OutboxRelay struct {
	db       *sql.DB
	producer *pkgkafka.Producer
	topic    string
	interval time.Duration
	logger   *slog.Logger
}

// New creates an OutboxRelay. topic is the Kafka topic to publish to.
func New(db *sql.DB, producer *pkgkafka.Producer, topic string, logger *slog.Logger) *OutboxRelay {
	return &OutboxRelay{
		db:       db,
		producer: producer,
		topic:    topic,
		interval: 2 * time.Second,
		logger:   logger,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.flush(ctx); err != nil && ctx.Err() == nil {
				r.logger.Error("outbox relay flush error", "error", err)
			}
		}
	}
}

func (r *OutboxRelay) flush(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, aggregate_id, aggregate_type, event_type, payload
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT 100
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []outboxRow
	for rows.Next() {
		var e outboxRow
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.AggregateType, &e.EventType, &e.Payload); err != nil {
			return err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range events {
		msg := pkgkafka.Message{
			Key:   []byte(e.AggregateID.String()),
			Value: e.Payload,
			Headers: map[string]string{
				"event_type":     e.EventType,
				"aggregate_type": e.AggregateType,
				"outbox_id":      e.ID.String(),
			},
		}
		if err := r.producer.Publish(ctx, msg); err != nil {
			r.logger.Error("outbox relay: failed to publish", "outbox_id", e.ID, "event_type", e.EventType, "error", err)
			// Continue — attempt the next event; this one will be retried next tick.
			continue
		}

		if _, err := r.db.ExecContext(ctx,
			`UPDATE outbox SET published_at = NOW() WHERE id = $1`, e.ID,
		); err != nil {
			r.logger.Error("outbox relay: failed to mark published", "outbox_id", e.ID, "error", err)
		}

		r.logger.Info("outbox relay: published event", "event_type", e.EventType, "aggregate_id", e.AggregateID)
	}
	return nil
}
