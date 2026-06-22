package usecases_test

import (
	"context"
	"io"
	"log/slog"
)

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type noopPublisher struct{}

func (n *noopPublisher) Publish(_ context.Context, _ string, _ []byte) error { return nil }

type capturePublisher struct {
	event   string
	payload []byte
}

func (c *capturePublisher) Publish(_ context.Context, event string, payload []byte) error {
	c.event = event
	c.payload = payload
	return nil
}
