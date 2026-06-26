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
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
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
	userRepo       contracts.UserRepository
	oauthRepo      contracts.OAuthRepository
	tokenRepo      contracts.RefreshTokenRepository
	authSvc        *AuthService
	googleConfig   *oauth2.Config
	facebookConfig *oauth2.Config
	cfg            *config.Config
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	userRepo contracts.UserRepository,
	oauthRepo contracts.OAuthRepository,
	tokenRepo contracts.RefreshTokenRepository,
	authSvc *AuthService,
	cfg *config.Config,
) *OAuthService {
	svc := &OAuthService{
		userRepo:  userRepo,
		oauthRepo: oauthRepo,
		tokenRepo: tokenRepo,
		authSvc:   authSvc,
		cfg:       cfg,
	}

	if cfg.GoogleClientID != "" {
		svc.googleConfig = &oauth2.Config{
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

	if cfg.FacebookClientID != "" {
		svc.facebookConfig = &oauth2.Config{
			ClientID:     cfg.FacebookClientID,
			ClientSecret: cfg.FacebookClientSecret,
			RedirectURL:  cfg.FacebookRedirectURL,
			Scopes:       []string{"email", "public_profile"},
			Endpoint:     facebook.Endpoint,
		}
	}

	return svc
}

// GetGoogleOAuthURL returns the authorization URL for Google OAuth2
func (s *OAuthService) GetGoogleOAuthURL(state string) (string, error) {
	if s.googleConfig == nil {
		return "", pkgerrors.NewInternal("OAUTH_NOT_CONFIGURED", "Google OAuth is not configured", nil)
	}
	return s.googleConfig.AuthCodeURL(state), nil
}

// GetFacebookOAuthURL returns the authorization URL for Facebook OAuth2
func (s *OAuthService) GetFacebookOAuthURL(state string) (string, error) {
	if s.facebookConfig == nil {
		return "", pkgerrors.NewInternal("OAUTH_NOT_CONFIGURED", "Facebook OAuth is not configured", nil)
	}
	return s.facebookConfig.AuthCodeURL(state), nil
}

// HandleGoogleCallback exchanges the authorization code for tokens and creates/updates the user
func (s *OAuthService) HandleGoogleCallback(ctx context.Context, code string) (*domain.User, *domain.RefreshToken, error) {
	if s.googleConfig == nil {
		return nil, nil, pkgerrors.NewInternal("OAUTH_NOT_CONFIGURED", "Google OAuth is not configured", nil)
	}

	token, err := s.googleConfig.Exchange(ctx, code)
	if err != nil {
		return nil, nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to exchange code: %v", err), err)
	}

	userInfo, err := s.getGoogleUserInfo(token)
	if err != nil {
		return nil, nil, err
	}

	return s.handleOAuthUser(ctx, userInfo)
}

// HandleFacebookCallback exchanges the authorization code for tokens and creates/updates the user
func (s *OAuthService) HandleFacebookCallback(ctx context.Context, code string) (*domain.User, *domain.RefreshToken, error) {
	if s.facebookConfig == nil {
		return nil, nil, pkgerrors.NewInternal("OAUTH_NOT_CONFIGURED", "Facebook OAuth is not configured", nil)
	}

	token, err := s.facebookConfig.Exchange(ctx, code)
	if err != nil {
		return nil, nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to exchange code: %v", err), err)
	}

	userInfo, err := s.getFacebookUserInfo(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	return s.handleOAuthUser(ctx, userInfo)
}

// handleOAuthUser creates or updates a user with an OAuth account
func (s *OAuthService) handleOAuthUser(ctx context.Context, userInfo *OAuthUserInfo) (*domain.User, *domain.RefreshToken, error) {
	oauthAccount, existingUser, err := s.oauthRepo.GetOAuthAccountByProviderUID(ctx, userInfo.Provider, userInfo.UID)
	if err == nil && oauthAccount != nil && existingUser != nil {
		refreshToken, err := s.generateRefreshTokenForUser(ctx, existingUser.ID)
		if err != nil {
			return nil, nil, err
		}
		return existingUser, refreshToken, nil
	}

	// Ignore not-found; any other error is real
	if err != nil {
		if appErr, ok := err.(*pkgerrors.AppError); !ok || appErr.Type != pkgerrors.NotFound {
			return nil, nil, err
		}
	}

	now := time.Now()
	user := &domain.User{
		ID:         uuid.New(),
		Email:      userInfo.Email,
		FullName:   userInfo.FullName,
		Role:       string(domain.RoleBuyer),
		IsVerified: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, nil, err
	}

	oauthAccount = &domain.OAuthAccount{
		ID:          uuid.New(),
		UserID:      user.ID,
		Provider:    userInfo.Provider,
		ProviderUID: userInfo.UID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.oauthRepo.CreateOAuthAccount(ctx, oauthAccount); err != nil {
		return nil, nil, err
	}

	refreshToken, err := s.generateRefreshTokenForUser(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

func (s *OAuthService) getGoogleUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to build request: %v", err), err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to get user info: %v", err), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to read response: %v", err), err)
	}

	var googleResp struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to parse user info: %v", err), err)
	}

	return &OAuthUserInfo{
		Email:    googleResp.Email,
		FullName: googleResp.Name,
		Provider: domain.GoogleProvider,
		UID:      googleResp.ID,
	}, nil
}

func (s *OAuthService) getFacebookUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://graph.facebook.com/me?fields=id,email,name", nil)
	if err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to build request: %v", err), err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to get user info: %v", err), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to read response: %v", err), err)
	}

	var facebookResp struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &facebookResp); err != nil {
		return nil, pkgerrors.NewInternal("OAUTH_FAILED", fmt.Sprintf("failed to parse user info: %v", err), err)
	}

	return &OAuthUserInfo{
		Email:    facebookResp.Email,
		FullName: facebookResp.Name,
		Provider: domain.FacebookProvider,
		UID:      facebookResp.ID,
	}, nil
}

func (s *OAuthService) generateRefreshTokenForUser(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	return s.authSvc.generateRefreshToken(ctx, userID)
}
