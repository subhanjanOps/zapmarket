package http

//go:generate mockgen -source=auth_service_iface.go -destination=../../mocks/auth_service_mock.go -package=mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// OAuthServicer is the subset of service.OAuthService the HTTP handlers depend on.
type OAuthServicer interface {
	GetGoogleOAuthURL(state string) (string, error)
	GetFacebookOAuthURL(state string) (string, error)
	HandleGoogleCallback(ctx context.Context, code string) (*domain.User, *domain.RefreshToken, error)
	HandleFacebookCallback(ctx context.Context, code string) (*domain.User, *domain.RefreshToken, error)
}

// AuthServicer is the subset of service.AuthService the HTTP handlers depend on.
// Holding an interface instead of the concrete struct enables testing with mocks.
type AuthServicer interface {
	RegisterUserPassword(ctx context.Context, email, password, fullName, role string) (*domain.User, *domain.RefreshToken, error)
	RegisterSeller(ctx context.Context, email, password, fullName string, profile domain.SellerProfile) (*domain.User, error)
	SendPhoneOTP(ctx context.Context, userID uuid.UUID, phone string) error
	VerifyPhoneOTP(ctx context.Context, userID uuid.UUID, code string) error
	UpdateRegistrationProfile(ctx context.Context, userID uuid.UUID, dob time.Time, gender, pfpURL *string, addr domain.Address, sellerProfile *domain.SellerProfile) error
	CompleteRegistration(ctx context.Context, userID uuid.UUID) error
	BootstrapAdmin(ctx context.Context, email, password, fullName string) (*domain.User, *domain.RefreshToken, error)
	LoginPassword(ctx context.Context, email, password string) (*domain.User, *domain.RefreshToken, error)
	RefreshAccessToken(ctx context.Context, refreshTokenString string) (string, error)
	ValidateAccessToken(ctx context.Context, tokenString string) (*domain.User, error)
	Logout(ctx context.Context, userID uuid.UUID, accessToken string) error
	BlacklistToken(ctx context.Context, tokenString string, ttl time.Duration)
	StoreOAuthState(ctx context.Context, state string) error
	ValidateAndConsumeOAuthState(ctx context.Context, state string) (bool, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	IssueRefreshTokenForUser(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	RequestPasswordReset(ctx context.Context, userEmail string) error
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
	SendEmailOTP(ctx context.Context, userEmail string) error
	VerifyEmailOTP(ctx context.Context, userID uuid.UUID, code string) error
	RequestPasswordResetOTP(ctx context.Context, phone string) error
	ResetPasswordWithOTP(ctx context.Context, phone, code, newPassword string) error
}
