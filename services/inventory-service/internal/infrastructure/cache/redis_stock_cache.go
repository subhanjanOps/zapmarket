package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned when a key is absent from the cache.
var ErrCacheMiss = errors.New("cache miss")

var luaReserve = goredis.NewScript(`
local available = redis.call('GET', KEYS[1])
if available == false then return -1 end
available = tonumber(available)
local qty = tonumber(ARGV[1])
if available < qty then return 0 end
return redis.call('DECRBY', KEYS[1], qty)
`)

var luaIncrIfExists = goredis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return -1 end
return redis.call('INCRBY', KEYS[1], ARGV[1])
`)

// RedisStockCache implements contracts.StockCachePort backed by go-redis.
type RedisStockCache struct {
	rdb *goredis.Client
}

func NewRedisStockCache(rdb *goredis.Client) *RedisStockCache {
	return &RedisStockCache{rdb: rdb}
}

func (c *RedisStockCache) StockKey(skuID uuid.UUID) string {
	return fmt.Sprintf("inv:stock:%s", skuID)
}

func (c *RedisStockCache) Get(ctx context.Context, key string) (int64, error) {
	v, err := c.rdb.Get(ctx, key).Int64()
	if errors.Is(err, goredis.Nil) {
		return 0, ErrCacheMiss
	}
	return v, err
}

func (c *RedisStockCache) Set(ctx context.Context, key string, value int64, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *RedisStockCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	exists, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, ErrCacheMiss
	}
	return c.rdb.IncrBy(ctx, key, delta).Result()
}

func (c *RedisStockCache) RunReserveScript(ctx context.Context, key string, qty int) (int64, error) {
	result, err := luaReserve.Run(ctx, c.rdb, []string{key}, strconv.Itoa(qty)).Int64()
	if errors.Is(err, goredis.Nil) {
		return -1, nil
	}
	return result, err
}

func (c *RedisStockCache) RunIncrIfExistsScript(ctx context.Context, key string, delta int) (int64, error) {
	result, err := luaIncrIfExists.Run(ctx, c.rdb, []string{key}, delta).Int64()
	if errors.Is(err, goredis.Nil) {
		return -1, nil
	}
	return result, err
}
