package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID
	ParentID  uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Product struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	SellerID    uuid.UUID
	Name        string
	Slug        string
	Description string
	Attributes  string
	Status      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type SKU struct {
	ID           uuid.UUID
	ProductID    uuid.UUID
	SkuCode      string
	VariantAttrs string
	PriceAmount  string
	Currency     string
	ComparePrice string
	WeightGrams  string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type ProductImage struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	SkuID     uuid.UUID
	Url       string
	position  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
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
