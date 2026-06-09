package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/auth-service/pkg/crypto"
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

// RegisterUserPassword registers a new user with email and password
func (s *AuthService) RegisterUserPassword(ctx context.Context, email, password, fullName string) (*domain.User, *domain.RefreshToken, error) {
	// Check if user already exists
	_, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, nil, domain.NewDomainError(domain.ErrUserExists, "user with this email already exists")
	}
	if _, ok := err.(*domain.DomainError); ok {
		if err.(*domain.DomainError).Type != domain.ErrUserNotFound {
			return nil, nil, err
		}
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to hash password: %v", err))
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &passwordHash,
		FullName:     fullName,
		Role:         "customer",
		IsVerified:   true, // Auto-verify for now; implement email verification in Phase 2
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	// Generate tokens
	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// LoginPassword authenticates a user with email and password
func (s *AuthService) LoginPassword(ctx context.Context, email, password string) (*domain.User, *domain.RefreshToken, error) {
	// Get user by email
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}

	// Verify password
	if user.PasswordHash == nil {
		return nil, nil, domain.NewDomainError(domain.ErrInvalidPassword, "this account uses OAuth login")
	}

	if !crypto.VerifyPassword(*user.PasswordHash, password) {
		return nil, nil, domain.NewDomainError(domain.ErrInvalidPassword, "invalid password")
	}

	// Generate tokens
	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// RefreshAccessToken validates a refresh token and issues a new access token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshTokenString string) (string, error) {
	// Validate refresh token structure
	claims, err := crypto.ValidateRefreshToken(refreshTokenString, s.cfg.JWTRefreshSecretKey)
	if err != nil {
		return "", err
	}

	// Verify token exists in database
	_, err = s.tokenRepo.GetRefreshTokenByTokenString(ctx, refreshTokenString)
	if err != nil {
		return "", err
	}

	// Get user to fetch latest role
	user, err := s.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return "", err
	}

	// Generate new access token
	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, s.cfg.JWTSecretKey, s.cfg.JWTAccessExpiryHours)
	if err != nil {
		return "", domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to generate access token: %v", err))
	}

	return accessToken, nil
}

// ValidateAccessToken verifies an access token and returns the user
func (s *AuthService) ValidateAccessToken(ctx context.Context, tokenString string) (*domain.User, error) {
	// Validate token signature and structure
	claims, err := crypto.ValidateAccessToken(tokenString, s.cfg.JWTSecretKey)
	if err != nil {
		return nil, err
	}

	// Fetch user from database
	user, err := s.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Verify user is not deleted
	if user.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "user has been deleted")
	}

	return user, nil
}

// Logout revokes all refresh tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	return s.tokenRepo.InvalidateUserTokens(ctx, userID)
}

// generateRefreshToken is a helper to create and store a refresh token
func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	tokenString, err := crypto.GenerateRefreshToken(userID, s.cfg.JWTRefreshSecretKey, s.cfg.JWTRefreshExpiryDays)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to generate refresh token: %v", err))
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTRefreshExpiryDays) * 24 * time.Hour)
	refreshToken, err := s.tokenRepo.CreateRefreshToken(ctx, userID, tokenString, expiresAt)
	if err != nil {
		return nil, err
	}

	// Store the raw token string for return
	refreshToken.TokenHash = tokenString // Override hash with actual token for return to client
	return refreshToken, nil
}

// GetUserByID retrieves a user by ID (for internal service-to-service calls)
func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

// GetUserByEmail retrieves a user by email (for internal use)
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.GetUserByEmail(ctx, email)
}
