package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// statusOnlyWriter captures just the status code for logging — never the body,
// which may contain tokens (engineering-standards: "Never log tokens").
type statusOnlyWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *statusOnlyWriter) WriteHeader(code int) {
	w.statusCode = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusOnlyWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// LoggingMiddleware logs method, path, status, and latency. Never logs the
// response body — it may contain tokens or PII.
func (h *Handler) LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := &statusOnlyWriter{ResponseWriter: w, statusCode: http.StatusOK}

		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		next(sw, r)

		slog.Info("HTTP response",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

// Handler wraps all HTTP handlers
type Handler struct {
	authSvc  AuthServicer
	oauthSvc OAuthServicer
	cfg      *config.Config
}

// NewHandler creates a new HTTP handler
func NewHandler(authSvc AuthServicer, oauthSvc OAuthServicer, cfg *config.Config) *Handler {
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
	// Account role: "buyer" or "seller"
	// example: buyer
	Role string `json:"role" example:"buyer"`
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
	// Role (buyer, seller, admin)
	Role string `json:"role" example:"buyer"`
	// Email verification status
	IsVerified bool `json:"is_verified" example:"true"`
	// Seller approval status (PENDING, APPROVED, SUSPENDED) — only present for seller role
	SellerStatus *string `json:"seller_status,omitempty" example:"PENDING"`
	// Creation timestamp
	CreatedAt string `json:"created_at" example:"2023-01-01T00:00:00Z"`
}

// writeResponse writes a JSON response. Delegates to pkg/httpx so the
// wire-level JSON plumbing isn't duplicated per service.
func (h *Handler) writeResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	httpx.JSON(w, statusCode, data)
}

// issueAccessToken generates a signed access token for user and returns it,
// writing an error response and returning ("", false) on failure.
func (h *Handler) issueAccessToken(w http.ResponseWriter, user *domain.User) (string, bool) {
	accessToken, err := crypto.GenerateAccessToken(user.ID, user.Email, user.Role, h.cfg.JWTSecretKey, h.cfg.JWTAccessExpiryHours)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate token")
		return "", false
	}
	return accessToken, true
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
	resp := &UserResponse{
		ID:         user.ID.String(),
		Email:      user.Email,
		Phone:      user.Phone,
		FullName:   user.FullName,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if user.Role == "seller" && user.SellerStatus != nil {
		resp.SellerStatus = user.SellerStatus
	}
	return resp
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
	if len(req.Email) > 254 {
		h.writeError(w, http.StatusBadRequest, "email must be 254 characters or fewer")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		h.writeError(w, http.StatusBadRequest, "email must be a valid email address")
		return
	}
	if len(req.Password) < 8 {
		h.writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if len(req.Password) > 72 {
		h.writeError(w, http.StatusBadRequest, "password must be 72 characters or fewer")
		return
	}
	if len(req.FullName) > 255 {
		h.writeError(w, http.StatusBadRequest, "full_name must be 255 characters or fewer")
		return
	}
	if req.Role != string(domain.RoleBuyer) && req.Role != string(domain.RoleSeller) {
		h.writeError(w, http.StatusBadRequest, "role must be 'buyer' or 'seller'")
		return
	}

	// Register user
	user, refreshToken, err := h.authSvc.RegisterUserPassword(r.Context(), req.Email, req.Password, req.FullName, req.Role)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	accessToken, ok := h.issueAccessToken(w, user)
	if !ok {
		return
	}

	h.writeResponse(w, http.StatusCreated, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
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
		pkgerrors.HandleHTTP(w, err)
		return
	}

	accessToken, ok := h.issueAccessToken(w, user)
	if !ok {
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
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
	token, ok := bearerToken(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}

	user, err := h.authSvc.ValidateAccessToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User: userToResponse(user),
	})
}

// AdminBootstrap handles POST /v1/auth/admin/bootstrap
// Creates the first admin user. Protected by ADMIN_BOOTSTRAP_SECRET env var.
// Returns 409 if an admin already exists (one-shot endpoint).
func (h *Handler) AdminBootstrap(w http.ResponseWriter, r *http.Request) {
	secret := h.cfg.AdminBootstrapSecret
	if secret == "" {
		h.writeError(w, http.StatusForbidden, "bootstrap not enabled — set ADMIN_BOOTSTRAP_SECRET")
		return
	}

	var req struct {
		Secret   string `json:"secret"`
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Secret != secret {
		h.writeError(w, http.StatusForbidden, "invalid bootstrap secret")
		return
	}
	if req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if req.FullName == "" {
		req.FullName = "System Admin"
	}

	user, refreshToken, err := h.authSvc.BootstrapAdmin(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	accessToken, ok := h.issueAccessToken(w, user)
	if !ok {
		return
	}

	h.writeResponse(w, http.StatusCreated, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	})
}

// Logout handles POST /auth/logout
// @Summary Logout
// @Description Revoke refresh tokens and blacklist the current access token
// @Tags auth
// @Security BearerAuth
// @Success 200 {object} map[string]string "Logged out"
// @Failure 401 {object} AuthResponse "Missing or invalid token"
// @Router /v1/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	accessToken, ok := bearerToken(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}

	if err := h.authSvc.Logout(r.Context(), uuid.Nil, accessToken); err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// bearerToken extracts the token from the "Authorization: Bearer <token>" header.
// Returns ("", false) when the header is absent or malformed.
func bearerToken(r *http.Request) (string, bool) {
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}

// generateState returns a cryptographically random 16-byte hex string for use
// as an OAuth state parameter to prevent CSRF.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
	state, err := generateState()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	if err := h.authSvc.StoreOAuthState(r.Context(), state); err != nil {
		slog.Warn("failed to store OAuth state", "error", err)
	}

	url, err := h.oauthSvc.GetGoogleOAuthURL(state)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
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
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "missing code parameter")
		return
	}
	if state == "" {
		h.writeError(w, http.StatusBadRequest, "missing state parameter")
		return
	}

	valid, err := h.authSvc.ValidateAndConsumeOAuthState(r.Context(), state)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to validate state")
		return
	}
	if !valid {
		h.writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	user, refreshToken, err := h.oauthSvc.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	accessToken, ok := h.issueAccessToken(w, user)
	if !ok {
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
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
	state, err := generateState()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	if err := h.authSvc.StoreOAuthState(r.Context(), state); err != nil {
		slog.Warn("failed to store OAuth state", "error", err)
	}

	url, err := h.oauthSvc.GetFacebookOAuthURL(state)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
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
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "missing code parameter")
		return
	}
	if state == "" {
		h.writeError(w, http.StatusBadRequest, "missing state parameter")
		return
	}

	valid, err := h.authSvc.ValidateAndConsumeOAuthState(r.Context(), state)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to validate state")
		return
	}
	if !valid {
		h.writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	user, refreshToken, err := h.oauthSvc.HandleFacebookCallback(r.Context(), code)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	accessToken, ok := h.issueAccessToken(w, user)
	if !ok {
		return
	}

	h.writeResponse(w, http.StatusOK, AuthResponse{
		User:         userToResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	})
}
