package cache

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisDeduplicator implements the consumer.deduplicator interface.
type RedisDeduplicator struct {
	rdb *goredis.Client
}

func NewRedisDeduplicator(rdb *goredis.Client) *RedisDeduplicator {
	return &RedisDeduplicator{rdb: rdb}
}

func (d *RedisDeduplicator) SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return d.rdb.SetNX(ctx, key, 1, ttl).Result()
}

func (d *RedisDeduplicator) Del(ctx context.Context, key string) error {
	return d.rdb.Del(ctx, key).Err()
}
