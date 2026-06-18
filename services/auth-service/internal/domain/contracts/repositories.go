// Package contracts defines the repository interfaces that internal/service
// depends on. Concrete implementations live in internal/repository; service
// code must depend only on these interfaces, never on the concrete structs.
package contracts

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// UserListParams defines filters for listing users.
type UserListParams struct {
	Role   string // empty = all roles
	Search string // partial match on email or full_name
	Limit  int
	Offset int
}

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	VerifyUser(ctx context.Context, userID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error

	// Admin operations
	ListUsers(ctx context.Context, params UserListParams) ([]*domain.User, int64, error)
	UpdateSellerStatus(ctx context.Context, userID uuid.UUID, status string) error
	ListSellers(ctx context.Context, status string, limit, offset int) ([]*domain.User, int64, error)
}

// OAuthRepository defines the interface for OAuth account persistence.
type OAuthRepository interface {
	CreateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error
	GetOAuthAccountByProviderUID(ctx context.Context, provider domain.OAuthProvider, providerUID string) (*domain.OAuthAccount, *domain.User, error)
	UpdateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error
	DeleteOAuthAccount(ctx context.Context, accountID uuid.UUID) error
}

// RefreshTokenRepository defines the interface for refresh token persistence.
type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*domain.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	GetRefreshTokenByTokenString(ctx context.Context, tokenString string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error
	InvalidateUserTokens(ctx context.Context, userID uuid.UUID) error
}
