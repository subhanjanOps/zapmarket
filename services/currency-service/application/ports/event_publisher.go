package ports

import "context"

// EventPublisher publishes domain events to an async transport (e.g. Kafka).
type EventPublisher interface {
	Publish(ctx context.Context, event string, payload []byte) error
}
