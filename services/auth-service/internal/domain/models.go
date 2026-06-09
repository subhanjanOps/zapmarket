package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user in the system
type User struct {
	ID           uuid.UUID
	Email        string
	Phone        *string
	PasswordHash *string // nil for OAuth-only accounts
	FullName     string
	Role         string // "customer", "seller", "admin"
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// OAuthProvider represents OAuth provider types
type OAuthProvider string

const (
	GoogleProvider   OAuthProvider = "google"
	FacebookProvider OAuthProvider = "facebook"
)

// OAuthAccount represents a linked OAuth provider account
type OAuthAccount struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Provider    OAuthProvider
	ProviderUID string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Address represents a user's shipping or billing address
type Address struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Label     *string // "home", "office", etc.
	Line1     string
	Line2     *string
	City      string
	State     string
	Country   string
	Pincode   string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NotificationPreference represents a user's notification settings
type NotificationPreference struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Channel   string // "email", "sms", "push"
	EventType string // "order.created", etc.
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Claims represents JWT claims
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Type      string    `json:"type"` // "access" or "refresh"
	ExpiresAt int64     `json:"exp"`
	IssuedAt  int64     `json:"iat"`
}

// Error types
type ErrorType string

const (
	ErrUserNotFound    ErrorType = "user_not_found"
	ErrInvalidPassword ErrorType = "invalid_password"
	ErrUserExists      ErrorType = "user_exists"
	ErrInvalidToken    ErrorType = "invalid_token"
	ErrExpiredToken    ErrorType = "expired_token"
	ErrOAuthFailed     ErrorType = "oauth_failed"
	ErrDatabaseError   ErrorType = "database_error"
)

// DomainError wraps domain-level errors
type DomainError struct {
	Type    ErrorType
	Message string
}

func (e *DomainError) Error() string {
	return string(e.Type) + ": " + e.Message
}

func NewDomainError(errType ErrorType, message string) *DomainError {
	return &DomainError{Type: errType, Message: message}
}
