// Package publisher provides a multi-topic Kafka publisher used by the
// order-management-service saga orchestrator.
package publisher

import (
	"context"
	"fmt"

	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

// Multi wraps one kafka.Producer per topic and exposes a single Publish method
// that routes by topic name.
type Multi struct {
	producers map[string]*pkgkafka.Producer
}

// NewMulti constructs a Multi publisher.  The caller is responsible for closing
// each producer when the process exits.
func NewMulti(producers map[string]*pkgkafka.Producer) *Multi {
	return &Multi{producers: producers}
}

// Publish routes the payload to the producer registered for topic.
func (m *Multi) Publish(ctx context.Context, topic, key string, payload []byte) error {
	p, ok := m.producers[topic]
	if !ok {
		return fmt.Errorf("publisher: no producer registered for topic %q", topic)
	}
	return p.Publish(ctx, pkgkafka.Message{Key: []byte(key), Value: payload})
}
