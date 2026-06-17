package kafka

// Message is a Kafka message exchanged between services.
type Message struct {
	// Key is used for partition routing (typically the aggregate ID).
	Key []byte
	// Value is the serialised event payload (JSON).
	Value []byte
	// Headers are optional key-value metadata pairs.
	Headers map[string]string
}
