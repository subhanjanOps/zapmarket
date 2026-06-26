package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned by NoopCache for all Get calls.
var ErrCacheMiss = errors.New("cache miss")

// NoopCache is a no-op implementation of contracts.OrderCache used when Redis
// is unavailable. Idempotency falls through to the database path.
type NoopCache struct{}

func NewNoopCache() *NoopCache { return &NoopCache{} }

func (c *NoopCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrCacheMiss
}

func (c *NoopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (c *NoopCache) SetNX(_ context.Context, _ string, _ string, _ time.Duration) (bool, error) {
	return true, nil
}

func (c *NoopCache) Del(_ context.Context, _ ...string) error {
	return nil
}
