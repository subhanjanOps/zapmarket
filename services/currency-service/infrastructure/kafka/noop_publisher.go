package kafka

import "context"

// NoopPublisher discards all events. Used when KAFKA_BROKERS is not set.
type NoopPublisher struct{}

func (n *NoopPublisher) Publish(_ context.Context, _ string, _ []byte) error { return nil }
