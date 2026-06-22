package kafka

import (
	"context"

	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

// KafkaPublisher adapts pkg/kafka.Producer to the ports.EventPublisher interface.
type KafkaPublisher struct {
	producer *pkgkafka.Producer
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	return &KafkaPublisher{producer: pkgkafka.NewProducer(brokers, topic)}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event string, payload []byte) error {
	return p.producer.Publish(ctx, pkgkafka.Message{
		Key:     []byte(event),
		Value:   payload,
		Headers: map[string]string{"event_type": event},
	})
}

func (p *KafkaPublisher) Close() error {
	return p.producer.Close()
}
