package http

// @title ZapMarket Auth Service API
// @version 1.0
// @description This is the authentication and user management service for ZapMarket.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /auth
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
	"github.com/zapmarket/zapmarket/services/auth-service/pkg/crypto"
)

// LoggingMiddleware logs request and response details
func (h *Handler) LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &strings.Builder{},
		}

		// Log request
		h.logRequest(r)

		// Process request
		next(lrw, r)

		// Log response
		h.logResponse(r, lrw.statusCode, time.Since(start), lrw.body.String())
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code and body
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	body        *strings.Builder
	wroteHeader bool
}

// WriteHeader captures the status code
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.wroteHeader = true
	lrw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if !lrw.wroteHeader {
		lrw.WriteHeader(http.StatusOK)
	}
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

// logRequest logs incoming request details
func (h *Handler) logRequest(r *http.Request) {
	var userID string

	// Try to extract user ID from JWT token if present
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			// token := parts[1]
			// We could validate the token here to get user ID, but for simplicity,
			// we'll just note that auth is present. In a real implementation,
			// you might want to extract claims without full validation for logging.
			userID = "[authenticated]"
		}
	}

	slog.Info("HTTP request",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_id", userID,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"referer", r.Referer(),
	)
}

// logResponse logs response details
func (h *Handler) logResponse(r *http.Request, status int, duration time.Duration, body string) {
	// Truncate body if too long for logging
	maxBodyLen := 500
	loggedBody := body
	if len(body) > maxBodyLen {
		loggedBody = body[:maxBodyLen] + "... (truncated)"
	}

	slog.Info("HTTP response",
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"response_body", loggedBody,
	)
}

// Handler wraps all HTTP handlers
type Handler struct {
	authSvc  *service.AuthService
	oauthSvc *service.OAuthService
	cfg      *config.Config
}

// NewHandler creates a new HTTP handler
func NewHandler(authSvc *service.AuthService, oauthSvc *service.OAuthService, cfg *config.Config) *Handler {
	return &Handler{
		authSvc:  authSvc,
		oauthSvc: oauthSvc,
		cfg:      cfg,
	}
}

// Request/Response types
type RegisterRequest struct {
	// User's email address
	// example: user@example.com
	Email string `json:"email" example:"user@example.com"`
	// User's password
	// example: secret123
	Password string `json:"password" example:"secret123"`
	// User's full name
	// example: John Doe
	FullName string `json:"full_name" example:"John Doe"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	// User's email address
	// example: user@example.com
	Email string `json:"email" example:"user@example.com"`
	// User's password
	// example: secret123
	Password string `json:"password" example:"secret123"`
}

// RefreshTokenRequest represents the refresh token request body
type RefreshTokenRequest struct {
	// Refresh token issued during login or registration
	// example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	// User object
	User *UserResponse `json:"user,omitempty"`
	// Access token for API authentication
	AccessToken string `json:"access_token,omitempty"`
	// Refresh token for obtaining new access tokens
	RefreshToken string `json:"refresh_token,omitempty"`
	// Error message if any
	Error string `json:"error,omitempty"`
	// HTTP status code
	Code int `json:"code,omitempty"`
}

// UserResponse represents a user in the system
type UserResponse struct {
	// Unique identifier
	ID string `json:"id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	// Email address
	Email string `json:"email" example:"user@example.com"`
	// Phone number (optional)
	Phone *string `json:"phone,omitempty" example:"+1234567890"`
	// Full name
	FullName string `json:"full_name" example:"John Doe"`
	// Role (customer, seller, admin)
	Role string `json:"role" example:"customer"`
	// Email verification status
	IsVerified bool `json:"is_verified" example:"true"`
	// Creation timestamp
	CreatedAt string `json:"created_at" example:"2023-01-01T00:00:00Z"`
}

// writeResponse writes a JSON response
func (h *Handler) writeResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, statusCode int, errMsg string) {
	h.writeResponse(w, statusCode, AuthResponse{
		Error: errMsg,
		Code:  statusCode,
	})
}

// userToResponse converts domain User to API response
func userToResponse(user *domain.User) *UserResponse {
	return &UserResponse{
		ID:         user.ID.String(),
		Email:      user.Email,
		Phone:      user.Phone,
		FullName:   user.FullName,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// Register handles POST /auth/register
// @Summary Register a new user
// @Description Register a new user with email, password, and full name
// @Tags auth
// @Accept json
// @Produce json
// @Param register body RegisterRequest true "User registration details"
// @Success 201 {object} AuthResponse "User registered successfully"
// @Failure 400 {object} AuthResponse "Invalid request body or missing fields"
// @Failure 409 {object} AuthResponse "User with this email already exists"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		h.writeError(w, http.StatusBadRequest, "email, password, and full_name are required")
		return
	}

	// Register user
	user, refreshToken, err := h.authSvc.RegisterUserPassword(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		domainErr, ok := err.(*domain.DomainError)
		if ok && domainErr.Type == domain.ErrUserExists {
			h.writeError(w, http.StatusConflict, domainErr.Message)
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate access token
	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, h.cfg.JWTSecretKey, h.cfg.JWTAccessExpiryHours)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	h.writeResponse(w, http.StatusCreated, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.TokenHash,
	})
}

// Login handles POST /auth/login
// @Summary Login a user
// @Description Authenticate a user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param login body LoginRequest true "User login credentials"
// @Success 200 {object} AuthResponse "Login successful"
// @Failure 400 {object} AuthResponse "Invalid request body or missing fields"
// @Failure 401 {object} AuthResponse "Invalid email or password"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Login user
	user, refreshToken, err := h.authSvc.LoginPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		domainErr, ok := err.(*domain.DomainError)
		if ok {
			if domainErr.Type == domain.ErrUserNotFound || domainErr.Type == domain.ErrInvalidPassword {
				h.writeError(w, http.StatusUnauthorized, "invalid email or password")
				return
			}
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate access token
	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, h.cfg.JWTSecretKey, h.cfg.JWTAccessExpiryHours)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.TokenHash,
	})
}

// Refresh handles POST /auth/refresh
// @Summary Refresh access token
// @Description Generate a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body RefreshTokenRequest true "Refresh token"
// @Security BearerAuth
// @Success 200 {object} AuthResponse "New access token generated"
// @Failure 400 {object} AuthResponse "Invalid request body or missing refresh token"
// @Failure 401 {object} AuthResponse "Invalid or expired refresh token"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		h.writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	// Generate new access token
	accessToken, err := h.authSvc.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		AccessToken: accessToken,
	})
}

// Me handles GET /auth/me (requires authentication)
// @Summary Get current user profile
// @Description Retrieve the profile of the currently authenticated user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AuthResponse "User profile retrieved"
// @Failure 401 {object} AuthResponse "Missing or invalid authorization header"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	// Remove "Bearer " prefix
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		h.writeError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	token := parts[1]

	// Validate token and get user
	user, err := h.authSvc.ValidateAccessToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User: userToResponse(user),
	})
}

// GoogleOAuthURL handles GET /auth/oauth/google/url
// @Summary Get Google OAuth URL
// @Description Get the URL for initiating Google OAuth flow
// @Tags oauth
// @Produce json
// @Param state query string false "State parameter for OAuth security"
// @Success 200 {object} map[string]string "OAuth URL"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /oauth/google/url [get]
func (h *Handler) GoogleOAuthURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		// Generate a simple random state
		state = "state-" + strings.ReplaceAll(time.Now().String(), " ", "-")[:16]
	}

	url, err := h.oauthSvc.GetGoogleOAuthURL(state)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{
		"oauth_url": url,
	})
}

// GoogleOAuthCallback handles GET /auth/oauth/google/callback?code=...&state=...
// @Summary Google OAuth callback
// @Description Handle the callback from Google OAuth service
// @Tags oauth
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Param state query string true "State parameter from Google"
// @Success 200 {object} AuthResponse "OAuth login successful"
// @Failure 400 {object} AuthResponse "Missing code parameter"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /oauth/google/callback [get]
func (h *Handler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	user, refreshToken, err := h.oauthSvc.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate access token
	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, h.cfg.JWTSecretKey, h.cfg.JWTAccessExpiryHours)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.TokenHash,
	})
}

// FacebookOAuthURL handles GET /auth/oauth/facebook/url
// @Summary Get Facebook OAuth URL
// @Description Get the URL for initiating Facebook OAuth flow
// @Tags oauth
// @Produce json
// @Param state query string false "State parameter for OAuth security"
// @Success 200 {object} map[string]string "OAuth URL"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /oauth/facebook/url [get]
func (h *Handler) FacebookOAuthURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		// Generate a simple random state
		state = "state-" + strings.ReplaceAll(time.Now().String(), " ", "-")[:16]
	}

	url, err := h.oauthSvc.GetFacebookOAuthURL(state)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{
		"oauth_url": url,
	})
}

// FacebookOAuthCallback handles GET /auth/oauth/facebook/callback?code=...&state=...
// @Summary Facebook OAuth callback
// @Description Handle the callback from Facebook OAuth service
// @Tags oauth
// @Produce json
// @Param code query string true "Authorization code from Facebook"
// @Param state query string true "State parameter from Facebook"
// @Success 200 {object} AuthResponse "OAuth login successful"
// @Failure 400 {object} AuthResponse "Missing code parameter"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /oauth/facebook/callback [get]
func (h *Handler) FacebookOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	user, refreshToken, err := h.oauthSvc.HandleFacebookCallback(r.Context(), code)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate access token
	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, h.cfg.JWTSecretKey, h.cfg.JWTAccessExpiryHours)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.TokenHash,
	})
}
