package redisstore

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisTokenBlacklist implements contracts.TokenBlacklist via Redis.
type RedisTokenBlacklist struct {
	rdb *goredis.Client
}

func NewTokenBlacklist(rdb *goredis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{rdb: rdb}
}

func (b *RedisTokenBlacklist) Add(ctx context.Context, tokenHash string, ttl time.Duration) error {
	return b.rdb.Set(ctx, "auth:blacklist:"+tokenHash, 1, ttl).Err()
}

func (b *RedisTokenBlacklist) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	n, err := b.rdb.Exists(ctx, "auth:blacklist:"+tokenHash).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// RedisOAuthStateStore implements contracts.OAuthStateStore via Redis.
type RedisOAuthStateStore struct {
	rdb *goredis.Client
}

func NewOAuthStateStore(rdb *goredis.Client) *RedisOAuthStateStore {
	return &RedisOAuthStateStore{rdb: rdb}
}

func (s *RedisOAuthStateStore) Store(ctx context.Context, state string, ttl time.Duration) error {
	return s.rdb.Set(ctx, "auth:oauth:state:"+state, 1, ttl).Err()
}

func (s *RedisOAuthStateStore) ConsumeAndValidate(ctx context.Context, state string) (bool, error) {
	n, err := s.rdb.Del(ctx, "auth:oauth:state:"+state).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
