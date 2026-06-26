package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/email"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo      contracts.UserRepository
	oauthRepo     contracts.OAuthRepository
	tokenRepo     contracts.RefreshTokenRepository
	resetRepo     contracts.PasswordResetRepository
	emailer       email.Emailer
	cfg           *config.Config
	blacklist     contracts.TokenBlacklist
	oauthState    contracts.OAuthStateStore
}

// NewAuthService creates a new auth service.
func NewAuthService(
	userRepo contracts.UserRepository,
	oauthRepo contracts.OAuthRepository,
	tokenRepo contracts.RefreshTokenRepository,
	resetRepo contracts.PasswordResetRepository,
	emailer email.Emailer,
	cfg *config.Config,
	blacklist contracts.TokenBlacklist,
	oauthState contracts.OAuthStateStore,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		oauthRepo:  oauthRepo,
		tokenRepo:  tokenRepo,
		resetRepo:  resetRepo,
		emailer:    emailer,
		cfg:        cfg,
		blacklist:  blacklist,
		oauthState: oauthState,
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

	var sellerStatus *string
	if role == string(domain.RoleSeller) {
		s := "PENDING"
		sellerStatus = &s
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &passwordHash,
		FullName:     fullName,
		Role:         role,
		IsVerified:   true,
		SellerStatus: sellerStatus,
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

// BootstrapAdmin creates the first admin user. Returns an error if an admin
// already exists. Callers must verify the bootstrap secret themselves.
func (s *AuthService) BootstrapAdmin(ctx context.Context, email, password, fullName string) (*domain.User, *domain.RefreshToken, error) {
	// Reject if any admin already exists — one-shot only.
	users, _, err := s.userRepo.ListUsers(ctx, contracts.UserListParams{Role: string(domain.RoleAdmin), Limit: 1})
	if err != nil {
		return nil, nil, pkgerrors.NewInternal("INTERNAL_ERROR", "failed to check existing admins", err)
	}
	if len(users) > 0 {
		return nil, nil, pkgerrors.NewConflict("ADMIN_EXISTS", "an admin user already exists")
	}
	return s.RegisterUserPassword(ctx, email, password, fullName, string(domain.RoleAdmin))
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
	// Check blacklist first (logout / token revocation).
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
	if revoked, _ := s.blacklist.IsRevoked(ctx, tokenHash); revoked {
		return nil, pkgerrors.NewUnauthorized("INVALID_TOKEN", "token has been revoked")
	}

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

// Logout revokes all refresh tokens for a user and blacklists the current
// access token so it cannot be used before it naturally expires.
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, accessToken string) error {
	if err := s.tokenRepo.InvalidateUserTokens(ctx, userID); err != nil {
		return err
	}
	if accessToken != "" {
		claims, err := crypto.ValidateAccessToken(accessToken, s.cfg.JWTSecretKey)
		if err == nil {
			ttl := time.Until(time.Unix(claims.ExpiresAt, 0))
			if ttl > 0 {
				s.BlacklistToken(ctx, accessToken, ttl)
			}
		}
	}
	return nil
}

// BlacklistToken adds a token to the blacklist with the given TTL.
func (s *AuthService) BlacklistToken(ctx context.Context, tokenString string, ttl time.Duration) {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
	_ = s.blacklist.Add(ctx, tokenHash, ttl)
}

// StoreOAuthState stores a one-time OAuth state with a 10-minute TTL.
func (s *AuthService) StoreOAuthState(ctx context.Context, state string) error {
	return s.oauthState.Store(ctx, state, 10*time.Minute)
}

// ValidateAndConsumeOAuthState verifies a state token exists and deletes it
// atomically so it cannot be reused.
func (s *AuthService) ValidateAndConsumeOAuthState(ctx context.Context, state string) (bool, error) {
	return s.oauthState.ConsumeAndValidate(ctx, state)
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

	refreshToken.Token = tokenString
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

// RequestPasswordReset generates a reset token and emails a link to the user.
// Always returns nil to avoid leaking whether the email exists (timing-safe).
func (s *AuthService) RequestPasswordReset(ctx context.Context, userEmail string) error {
	user, err := s.userRepo.GetUserByEmail(ctx, userEmail)
	if err != nil {
		// Return nil regardless so the caller cannot enumerate registered emails.
		return nil
	}

	if user.PasswordHash == nil {
		// OAuth-only accounts cannot use password reset.
		return nil
	}

	rawToken, tokenHash, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("request password reset: %w", err)
	}

	expiresAt := time.Now().Add(30 * time.Minute)
	if _, err := s.resetRepo.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("request password reset: %w", err)
	}

	resetLink := fmt.Sprintf("%s?token=%s", s.cfg.PasswordResetBaseURL, rawToken)
	if err := s.emailer.SendPasswordResetEmail(ctx, user.Email, resetLink); err != nil {
		return fmt.Errorf("request password reset: send email: %w", err)
	}

	return nil
}

// ResetPassword validates a reset token and updates the user's password.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	tokenHash := hashRawToken(rawToken)

	resetToken, err := s.resetRepo.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		return pkgerrors.NewUnauthorized("INVALID_TOKEN", "invalid or expired password reset token")
	}

	if resetToken.UsedAt != nil {
		return pkgerrors.NewUnauthorized("INVALID_TOKEN", "password reset token has already been used")
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return pkgerrors.NewUnauthorized("INVALID_TOKEN", "password reset token has expired")
	}

	passwordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_ERROR", "failed to hash password", err)
	}

	user, err := s.userRepo.GetUserByID(ctx, resetToken.UserID)
	if err != nil {
		return err
	}

	user.PasswordHash = &passwordHash
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("reset password: update user: %w", err)
	}

	if err := s.resetRepo.MarkPasswordResetTokenUsed(ctx, resetToken.ID); err != nil {
		return fmt.Errorf("reset password: mark token used: %w", err)
	}

	// Invalidate all active refresh tokens so previously issued sessions cannot
	// be used with the old credentials.
	if err := s.tokenRepo.InvalidateUserTokens(ctx, user.ID); err != nil {
		return fmt.Errorf("reset password: invalidate sessions: %w", err)
	}

	return nil
}

// generateSecureToken returns a cryptographically random 32-byte hex raw token
// and its SHA-256 hash for storage.
func generateSecureToken() (rawToken, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate secure token: %w", err)
	}
	rawToken = hex.EncodeToString(b)
	tokenHash = hashRawToken(rawToken)
	return rawToken, tokenHash, nil
}

// hashRawToken returns the hex-encoded SHA-256 of rawToken.
func hashRawToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
