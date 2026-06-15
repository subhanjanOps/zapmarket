package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/repository"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo  *repository.UserRepository
	oauthRepo *repository.OAuthRepository
	tokenRepo *repository.RefreshTokenRepository
	cfg       *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *repository.UserRepository,
	oauthRepo *repository.OAuthRepository,
	tokenRepo *repository.RefreshTokenRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		oauthRepo: oauthRepo,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}
}

// RegisterUserPassword registers a new user with email, password, and role.
// role must be domain.RoleBuyer or domain.RoleSeller; domain.RoleAdmin is not self-assignable.
func (s *AuthService) RegisterUserPassword(ctx context.Context, email, password, fullName, role string) (*domain.User, *domain.RefreshToken, error) {
	_, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, nil, pkgerrors.NewConflict("USER_ALREADY_EXISTS", "user with this email already exists")
	}
	appErr, ok := err.(*pkgerrors.AppError)
	if !ok || appErr.Type != pkgerrors.NotFound {
		return nil, nil, err
	}

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, nil, pkgerrors.NewInternal("INTERNAL_ERROR", fmt.Sprintf("failed to hash password: %v", err), err)
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &passwordHash,
		FullName:     fullName,
		Role:         role,
		IsVerified:   true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, nil, err
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// LoginPassword authenticates a user with email and password
func (s *AuthService) LoginPassword(ctx context.Context, email, password string) (*domain.User, *domain.RefreshToken, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Return Unauthorized regardless of whether the email exists (security)
		return nil, nil, pkgerrors.NewUnauthorized("INVALID_CREDENTIALS", "invalid email or password")
	}

	if user.PasswordHash == nil {
		return nil, nil, pkgerrors.NewUnauthorized("INVALID_CREDENTIALS", "this account uses OAuth login")
	}

	if !crypto.VerifyPassword(*user.PasswordHash, password) {
		return nil, nil, pkgerrors.NewUnauthorized("INVALID_CREDENTIALS", "invalid email or password")
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// RefreshAccessToken validates a refresh token and issues a new access token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshTokenString string) (string, error) {
	claims, err := crypto.ValidateRefreshToken(refreshTokenString, s.cfg.JWTRefreshSecretKey)
	if err != nil {
		return "", pkgerrors.NewUnauthorized("INVALID_TOKEN", "invalid or expired refresh token")
	}

	_, err = s.tokenRepo.GetRefreshTokenByTokenString(ctx, refreshTokenString)
	if err != nil {
		return "", err
	}

	user, err := s.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return "", err
	}

	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, s.cfg.JWTSecretKey, s.cfg.JWTAccessExpiryHours)
	if err != nil {
		return "", pkgerrors.NewInternal("INTERNAL_ERROR", fmt.Sprintf("failed to generate access token: %v", err), err)
	}

	return accessToken, nil
}

// ValidateAccessToken verifies an access token and returns the user
func (s *AuthService) ValidateAccessToken(ctx context.Context, tokenString string) (*domain.User, error) {
	claims, err := crypto.ValidateAccessToken(tokenString, s.cfg.JWTSecretKey)
	if err != nil {
		return nil, pkgerrors.NewUnauthorized("INVALID_TOKEN", "invalid or expired token")
	}

	user, err := s.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if user.DeletedAt != nil {
		return nil, pkgerrors.NewUnauthorized("INVALID_TOKEN", "user has been deleted")
	}

	return user, nil
}

// Logout revokes all refresh tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	return s.tokenRepo.InvalidateUserTokens(ctx, userID)
}

// generateRefreshToken creates and stores a refresh token
func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	tokenString, err := crypto.GenerateRefreshToken(userID, s.cfg.JWTRefreshSecretKey, s.cfg.JWTRefreshExpiryDays)
	if err != nil {
		return nil, pkgerrors.NewInternal("INTERNAL_ERROR", fmt.Sprintf("failed to generate refresh token: %v", err), err)
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTRefreshExpiryDays) * 24 * time.Hour)
	refreshToken, err := s.tokenRepo.CreateRefreshToken(ctx, userID, tokenString, expiresAt)
	if err != nil {
		return nil, err
	}

	refreshToken.TokenHash = tokenString
	return refreshToken, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

// GetUserByEmail retrieves a user by email
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.GetUserByEmail(ctx, email)
}
