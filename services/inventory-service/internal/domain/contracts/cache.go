package contracts

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StockCachePort defines cache operations the inventory service needs.
// Implementations must guarantee that RunReserveScript and RunIncrIfExistsScript
// are executed atomically (e.g. via Redis Lua scripts).
type StockCachePort interface {
	// Get returns the cached available qty for a SKU. Returns (0, ErrCacheMiss) if absent.
	Get(ctx context.Context, key string) (int64, error)
	// Set stores the available qty with the given TTL.
	Set(ctx context.Context, key string, value int64, ttl time.Duration) error
	// IncrBy atomically increments the value at key by delta, returning the new value.
	// Returns (0, ErrCacheMiss) if the key does not exist.
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	// RunReserveScript atomically checks available qty and decrements.
	// Returns: -1 = cache miss, 0 = insufficient, >0 = remaining after decrement.
	RunReserveScript(ctx context.Context, key string, qty int) (int64, error)
	// RunIncrIfExistsScript increments by delta only if key exists.
	// Returns -1 if the key was absent.
	RunIncrIfExistsScript(ctx context.Context, key string, delta int) (int64, error)
	// Delete removes the key from the cache. Used as a last-resort rollback
	// when an increment fails, so the next read falls through to Postgres.
	Delete(ctx context.Context, key string) error
	// StockKey returns the cache key for a given SKU ID.
	StockKey(skuID uuid.UUID) string
}
