package redisstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
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

// RedisMFASessionStore stores short-lived MFA session tokens in Redis.
type RedisMFASessionStore struct {
	rdb *goredis.Client
}

func NewMFASessionStore(rdb *goredis.Client) *RedisMFASessionStore {
	return &RedisMFASessionStore{rdb: rdb}
}

func (s *RedisMFASessionStore) SetMFASession(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return s.rdb.Set(ctx, "auth:mfa:session:"+token, userID.String(), ttl).Err()
}

func (s *RedisMFASessionStore) GetMFASession(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := s.rdb.Get(ctx, "auth:mfa:session:"+token).Result()
	if errors.Is(err, goredis.Nil) {
		return uuid.Nil, errors.New("not found")
	}
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(val)
}

func (s *RedisMFASessionStore) DeleteMFASession(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, "auth:mfa:session:"+token).Err()
}

// NoopMFASessionStore is a fallback when Redis is unavailable.
type NoopMFASessionStore struct{}

func (s *NoopMFASessionStore) SetMFASession(_ context.Context, _ string, _ uuid.UUID, _ time.Duration) error {
	return errors.New("MFA requires Redis")
}
func (s *NoopMFASessionStore) GetMFASession(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, errors.New("MFA requires Redis")
}
func (s *NoopMFASessionStore) DeleteMFASession(_ context.Context, _ string) error { return nil }
