package cache

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisIdempotencyCache implements contracts.IdempotencyCache via Redis.
type RedisIdempotencyCache struct {
	rdb *goredis.Client
}

func NewRedisIdempotencyCache(rdb *goredis.Client) *RedisIdempotencyCache {
	return &RedisIdempotencyCache{rdb: rdb}
}

func (c *RedisIdempotencyCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *RedisIdempotencyCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *RedisIdempotencyCache) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

func (c *RedisIdempotencyCache) Del(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// NoopIdempotencyCache is used when Redis is unavailable; idempotency falls
// through to the DB path (correct but not cached).
type NoopIdempotencyCache struct{}

var ErrCacheMiss = errors.New("cache miss")

func (c *NoopIdempotencyCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrCacheMiss
}
func (c *NoopIdempotencyCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (c *NoopIdempotencyCache) SetNX(_ context.Context, _ string, _ string, _ time.Duration) (bool, error) {
	return true, nil
}
func (c *NoopIdempotencyCache) Del(_ context.Context, _ string) error { return nil }
