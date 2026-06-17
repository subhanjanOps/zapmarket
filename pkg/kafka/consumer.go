package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// HandlerFunc is called for each message received from Kafka.
// Return a non-nil error to signal that the message should not be committed.
type HandlerFunc func(ctx context.Context, msg Message) error

// Consumer reads messages from a single Kafka topic using a consumer group.
type Consumer struct {
	reader *kafkago.Reader
}

// NewConsumer creates a Consumer for the given brokers, topic, and group ID.
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: time.Second,
	})
	return &Consumer{reader: r}
}

// Run blocks and calls handler for every message. It commits the offset only
// after handler returns nil. Stops when ctx is cancelled.
func (c *Consumer) Run(ctx context.Context, handler HandlerFunc) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		msg := Message{
			Key:     m.Key,
			Value:   m.Value,
			Headers: make(map[string]string, len(m.Headers)),
		}
		for _, h := range m.Headers {
			msg.Headers[h.Key] = string(h.Value)
		}

		if err := handler(ctx, msg); err != nil {
			// Skip commit — message will be redelivered.
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

// Close shuts down the reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
