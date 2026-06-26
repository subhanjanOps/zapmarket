package contracts

import (
	"context"
	"time"
)

// OrderCache defines the cache operations the order service needs.
type OrderCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
}
