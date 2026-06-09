package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/repository"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

// OAuthUserInfo holds user information from OAuth providers
type OAuthUserInfo struct {
	Email    string
	FullName string
	Provider domain.OAuthProvider
	UID      string
}

// OAuthService handles OAuth authentication logic
type OAuthService struct {
	userRepo       *repository.UserRepository
	oauthRepo      *repository.OAuthRepository
	tokenRepo      *repository.RefreshTokenRepository
	googleConfig   *oauth2.Config
	facebookConfig *oauth2.Config
	cfg            *config.Config
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	userRepo *repository.UserRepository,
	oauthRepo *repository.OAuthRepository,
	tokenRepo *repository.RefreshTokenRepository,
	cfg *config.Config,
) *OAuthService {
	service := &OAuthService{
		userRepo:  userRepo,
		oauthRepo: oauthRepo,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}

	// Initialize Google OAuth2 config
	if cfg.GoogleClientID != "" {
		service.googleConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		}
	}

	// Initialize Facebook OAuth2 config
	if cfg.FacebookClientID != "" {
		service.facebookConfig = &oauth2.Config{
			ClientID:     cfg.FacebookClientID,
			ClientSecret: cfg.FacebookClientSecret,
			RedirectURL:  cfg.FacebookRedirectURL,
			Scopes:       []string{"email", "public_profile"},
			Endpoint:     facebook.Endpoint,
		}
	}

	return service
}

// GetGoogleOAuthURL returns the authorization URL for Google OAuth2
func (s *OAuthService) GetGoogleOAuthURL(state string) (string, error) {
	if s.googleConfig == nil {
		return "", domain.NewDomainError(domain.ErrOAuthFailed, "Google OAuth is not configured")
	}
	return s.googleConfig.AuthCodeURL(state), nil
}

// GetFacebookOAuthURL returns the authorization URL for Facebook OAuth2
func (s *OAuthService) GetFacebookOAuthURL(state string) (string, error) {
	if s.facebookConfig == nil {
		return "", domain.NewDomainError(domain.ErrOAuthFailed, "Facebook OAuth is not configured")
	}
	return s.facebookConfig.AuthCodeURL(state), nil
}

// HandleGoogleCallback exchanges the authorization code for tokens and creates/updates the user
func (s *OAuthService) HandleGoogleCallback(ctx context.Context, code string) (*domain.User, *domain.RefreshToken, error) {
	if s.googleConfig == nil {
		return nil, nil, domain.NewDomainError(domain.ErrOAuthFailed, "Google OAuth is not configured")
	}

	// Exchange code for token
	token, err := s.googleConfig.Exchange(ctx, code)
	if err != nil {
		return nil, nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to exchange code: %v", err))
	}

	// Get user info
	userInfo, err := s.getGoogleUserInfo(token)
	if err != nil {
		return nil, nil, err
	}

	return s.handleOAuthUser(ctx, userInfo)
}

// HandleFacebookCallback exchanges the authorization code for tokens and creates/updates the user
func (s *OAuthService) HandleFacebookCallback(ctx context.Context, code string) (*domain.User, *domain.RefreshToken, error) {
	if s.facebookConfig == nil {
		return nil, nil, domain.NewDomainError(domain.ErrOAuthFailed, "Facebook OAuth is not configured")
	}

	// Exchange code for token
	token, err := s.facebookConfig.Exchange(ctx, code)
	if err != nil {
		return nil, nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to exchange code: %v", err))
	}

	// Get user info
	userInfo, err := s.getFacebookUserInfo(token)
	if err != nil {
		return nil, nil, err
	}

	return s.handleOAuthUser(ctx, userInfo)
}

// handleOAuthUser creates or updates a user with OAuth account
func (s *OAuthService) handleOAuthUser(ctx context.Context, userInfo *OAuthUserInfo) (*domain.User, *domain.RefreshToken, error) {
	// Try to find existing OAuth account
	oauthAccount, existingUser, err := s.oauthRepo.GetOAuthAccountByProviderUID(ctx, userInfo.Provider, userInfo.UID)
	if err == nil && oauthAccount != nil && existingUser != nil {
		// User already exists, just generate tokens
		refreshToken, err := s.generateRefreshTokenForUser(ctx, existingUser.ID)
		if err != nil {
			return nil, nil, err
		}
		return existingUser, refreshToken, nil
	}

	// Check if domain error is NOT "user not found"
	if err != nil {
		if domainErr, ok := err.(*domain.DomainError); ok && domainErr.Type != domain.ErrUserNotFound {
			return nil, nil, err
		}
	}

	// Create new user and OAuth account
	now := time.Now()
	user := &domain.User{
		ID:         uuid.New(),
		Email:      userInfo.Email,
		FullName:   userInfo.FullName,
		Role:       "customer",
		IsVerified: true, // OAuth email is already verified by provider
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Create user
	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	// Create OAuth account
	oauthAccount = &domain.OAuthAccount{
		ID:          uuid.New(),
		UserID:      user.ID,
		Provider:    userInfo.Provider,
		ProviderUID: userInfo.UID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = s.oauthRepo.CreateOAuthAccount(ctx, oauthAccount)
	if err != nil {
		return nil, nil, err
	}

	// Generate tokens
	refreshToken, err := s.generateRefreshTokenForUser(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// getGoogleUserInfo fetches user information from Google
func (s *OAuthService) getGoogleUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to get user info: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to read response: %v", err))
	}

	// Parse JSON manually to extract id, email, name
	// For production, use a JSON decoder
	var googleResp struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	err = json.Unmarshal(body, &googleResp)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to parse user info: %v", err))
	}

	return &OAuthUserInfo{
		Email:    googleResp.Email,
		FullName: googleResp.Name,
		Provider: domain.GoogleProvider,
		UID:      googleResp.ID,
	}, nil
}

// getFacebookUserInfo fetches user information from Facebook
func (s *OAuthService) getFacebookUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	resp, err := http.Get("https://graph.facebook.com/me?fields=id,email,name&access_token=" + token.AccessToken)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to get user info: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to read response: %v", err))
	}

	// Parse JSON manually to extract id, email, name
	var facebookResp struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	err = json.Unmarshal(body, &facebookResp)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrOAuthFailed, fmt.Sprintf("failed to parse user info: %v", err))
	}

	return &OAuthUserInfo{
		Email:    facebookResp.Email,
		FullName: facebookResp.Name,
		Provider: domain.FacebookProvider,
		UID:      facebookResp.ID,
	}, nil
}

// generateRefreshTokenForUser is a helper to create and store a refresh token
func (s *OAuthService) generateRefreshTokenForUser(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	// This mirrors the logic in AuthService but avoids duplication
	// In production, extract this to a shared helper
	authSvc := NewAuthService(s.userRepo, s.oauthRepo, s.tokenRepo, s.cfg)
	return authSvc.generateRefreshToken(ctx, userID)
}
