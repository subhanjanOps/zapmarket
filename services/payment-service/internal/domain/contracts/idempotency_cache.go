package contracts

import (
	"context"
	"time"
)

// IdempotencyCache defines the cache operations the payment service needs for
// idempotency. Concrete implementations may be backed by Redis or a no-op fallback.
type IdempotencyCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// SetNX sets the key only if it does not exist. Returns (true, nil) when acquired.
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, key string) error
}
