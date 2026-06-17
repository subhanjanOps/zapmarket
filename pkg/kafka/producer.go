package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer publishes messages to a Kafka topic.
type Producer struct {
	writer *kafkago.Writer
}

// NewProducer creates a Producer that writes to the given brokers and topic.
func NewProducer(brokers []string, topic string) *Producer {
	w := &kafkago.Writer{
		Addr:                   kafkago.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafkago.LeastBytes{},
		WriteTimeout:           10 * time.Second,
		RequiredAcks:           kafkago.RequireOne,
		AllowAutoTopicCreation: true,
	}
	return &Producer{writer: w}
}

// Publish sends a message to Kafka.
func (p *Producer) Publish(ctx context.Context, msg Message) error {
	km := kafkago.Message{
		Key:   msg.Key,
		Value: msg.Value,
	}
	for k, v := range msg.Headers {
		km.Headers = append(km.Headers, kafkago.Header{Key: k, Value: []byte(v)})
	}
	return p.writer.WriteMessages(ctx, km)
}

// Close shuts down the writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
