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
	GetUserByPhone(ctx context.Context, phone string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, dob *time.Time, gender, pfpURL *string, phoneVerified *bool, registrationStep *int) error
	CompleteRegistration(ctx context.Context, userID uuid.UUID) error
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

// PreferencesRepository stores per-user key-value preferences.
type PreferencesRepository interface {
	Get(ctx context.Context, userID uuid.UUID, key string) (string, bool, error)
	Set(ctx context.Context, userID uuid.UUID, key, value string) error
}

// PasswordResetRepository defines the interface for password reset token persistence.
type PasswordResetRepository interface {
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.PasswordResetToken, error)
	GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error
}

// OTPRepository defines the interface for OTP verification persistence.
type OTPRepository interface {
	CreateOTP(ctx context.Context, otp *domain.OTPVerification) error
	GetLatestUnusedOTP(ctx context.Context, userID uuid.UUID, purpose domain.OTPPurpose) (*domain.OTPVerification, error)
	MarkOTPUsed(ctx context.Context, otpID uuid.UUID) error
}

// RefreshTokenRepository defines the interface for refresh token persistence.
type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*domain.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	GetRefreshTokenByTokenString(ctx context.Context, tokenString string) (*domain.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, tokenID uuid.UUID, newToken string, expiresAt time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error
	InvalidateUserTokens(ctx context.Context, userID uuid.UUID) error
}

// SellerProfileRepository defines persistence for seller onboarding data.
type SellerProfileRepository interface {
	Create(ctx context.Context, profile *domain.SellerProfile) error
	Update(ctx context.Context, profile *domain.SellerProfile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.SellerProfile, error)
}

// AddressRepository defines persistence for user addresses.
type AddressRepository interface {
	Create(ctx context.Context, addr *domain.Address) error
	GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*domain.Address, error)
}
