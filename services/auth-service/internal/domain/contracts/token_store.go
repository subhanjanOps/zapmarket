package contracts

import (
	"context"
	"time"
)

// TokenBlacklist is the port the auth service uses to blacklist revoked tokens.
// Concrete implementations may be backed by Redis or a no-op fallback.
type TokenBlacklist interface {
	// Add marks token as revoked until ttl elapses.
	Add(ctx context.Context, tokenHash string, ttl time.Duration) error
	// IsRevoked returns true if the token has been previously revoked.
	IsRevoked(ctx context.Context, tokenHash string) (bool, error)
}

// OAuthStateStore is the port for storing and consuming one-time OAuth CSRF state tokens.
type OAuthStateStore interface {
	// Store persists state for ttl. Returns nil when storage is unavailable.
	Store(ctx context.Context, state string, ttl time.Duration) error
	// ConsumeAndValidate atomically deletes state and returns whether it existed.
	ConsumeAndValidate(ctx context.Context, state string) (bool, error)
}
