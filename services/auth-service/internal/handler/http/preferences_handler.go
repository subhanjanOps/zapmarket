package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
)

var validCurrencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// prefsUserIDKey is the context key used to inject a user ID for testing.
// In production the handler extracts the user from the Authorization header.
type prefsUserIDKeyType struct{}

// PrefsUserIDKey can be used in tests to inject a uuid.UUID into context,
// bypassing JWT validation.
var PrefsUserIDKey prefsUserIDKeyType

// PreferencesHandler handles GET/PUT /v1/users/me/preferences.
type PreferencesHandler struct {
	repo    contracts.PreferencesRepository
	authSvc *service.AuthService
}

// NewPreferencesHandler creates a new PreferencesHandler.
// authSvc may be nil only in unit tests that inject a user via PrefsUserIDKey.
func NewPreferencesHandler(repo contracts.PreferencesRepository, authSvc *service.AuthService) *PreferencesHandler {
	return &PreferencesHandler{repo: repo, authSvc: authSvc}
}

// resolveUserID extracts the authenticated user's ID from the request.
// It first checks the test-injection context key, then falls back to JWT validation.
func (h *PreferencesHandler) resolveUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	// Test injection path
	if id, ok := r.Context().Value(PrefsUserIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return id, true
	}

	// Production path: validate Bearer token
	header := r.Header.Get("Authorization")
	if header == "" {
		httpx.Error(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header is required")
		return uuid.Nil, false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "bearer token required")
		return uuid.Nil, false
	}
	user, err := h.authSvc.ValidateAccessToken(r.Context(), strings.TrimSpace(parts[1]))
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")
		return uuid.Nil, false
	}
	return user.ID, true
}

// GetPreferences handles GET /v1/users/me/preferences
func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	val, found, err := h.repo.Get(r.Context(), userID, "display_currency")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	resp := map[string]string{}
	if found {
		resp["display_currency"] = val
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// SetPreferences handles PUT /v1/users/me/preferences
func (h *PreferencesHandler) SetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		DisplayCurrency string `json:"display_currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if !validCurrencyCode.MatchString(body.DisplayCurrency) {
		httpx.Error(w, http.StatusBadRequest, "INVALID_CURRENCY", "display_currency must be a 3-letter uppercase ISO code (e.g. EUR)")
		return
	}

	if err := h.repo.Set(r.Context(), userID, "display_currency", body.DisplayCurrency); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"display_currency": body.DisplayCurrency})
}
