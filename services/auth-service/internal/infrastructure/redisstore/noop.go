package redisstore

import (
	"context"
	"time"
)

// NoopTokenBlacklist is used when Redis is unavailable; blacklisting is a no-op
// so short-lived tokens expire naturally without blocking service startup.
type NoopTokenBlacklist struct{}

func (n *NoopTokenBlacklist) Add(_ context.Context, _ string, _ time.Duration) error { return nil }
func (n *NoopTokenBlacklist) IsRevoked(_ context.Context, _ string) (bool, error)     { return false, nil }

// NoopOAuthStateStore degrades CSRF protection when Redis is unavailable.
// It always reports states as valid (no verification possible without storage).
type NoopOAuthStateStore struct{}

func (n *NoopOAuthStateStore) Store(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (n *NoopOAuthStateStore) ConsumeAndValidate(_ context.Context, _ string) (bool, error) {
	return true, nil
}
