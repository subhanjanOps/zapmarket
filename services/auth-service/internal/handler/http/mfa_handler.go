package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	pkgcrypto "github.com/zapmarket/zapmarket/pkg/crypto"
	pkgcfg "github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// MFAServicer is the subset of service.MFAService the HTTP handlers depend on.
type MFAServicer interface {
	Enroll(ctx context.Context, userID uuid.UUID, email string) (otpauthURL string, backupCodes []string, err error)
	VerifyEnrollment(ctx context.Context, userID uuid.UUID, code string) error
	Disable(ctx context.Context, userID uuid.UUID, code string) error
	Challenge(ctx context.Context, mfaToken, totpCode string) (uuid.UUID, error)
}

// MFAHandler handles TOTP MFA endpoints.
type MFAHandler struct {
	mfa  MFAServicer
	auth AuthServicer
	cfg  *pkgcfg.Config
}

func NewMFAHandler(mfa MFAServicer, auth AuthServicer, cfg *pkgcfg.Config) *MFAHandler {
	return &MFAHandler{mfa: mfa, auth: auth, cfg: cfg}
}

// EnrollMFA godoc
// @Summary Start TOTP enrollment
// @Tags mfa
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/mfa/enroll [post]
func (h *MFAHandler) EnrollMFA(w http.ResponseWriter, r *http.Request) {
	tok, ok := bearerToken(r)
	if !ok {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "UNAUTHORIZED"})
		return
	}
	user, err := h.auth.ValidateAccessToken(r.Context(), tok)
	if err != nil {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "UNAUTHORIZED"})
		return
	}
	url, codes, err := h.mfa.Enroll(r.Context(), user.ID, user.Email)
	if err != nil {
		mfaWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	mfaWriteJSON(w, http.StatusOK, map[string]interface{}{
		"otpauth_url":  url,
		"backup_codes": codes,
	})
}

// VerifyEnrollment godoc
// @Summary Confirm TOTP enrollment with first code
// @Tags mfa
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 204
// @Router /auth/mfa/verify-enrollment [post]
func (h *MFAHandler) VerifyEnrollment(w http.ResponseWriter, r *http.Request) {
	tok, ok := bearerToken(r)
	if !ok {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "UNAUTHORIZED"})
		return
	}
	user, err := h.auth.ValidateAccessToken(r.Context(), tok)
	if err != nil {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "UNAUTHORIZED"})
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		mfaWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "INVALID_REQUEST"})
		return
	}
	if err := h.mfa.VerifyEnrollment(r.Context(), user.ID, body.Code); err != nil {
		mfaWriteJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DisableMFA godoc
// @Summary Disable TOTP for the current user
// @Tags mfa
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 204
// @Router /auth/mfa/disable [post]
func (h *MFAHandler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	tok, ok := bearerToken(r)
	if !ok {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "UNAUTHORIZED"})
		return
	}
	user, err := h.auth.ValidateAccessToken(r.Context(), tok)
	if err != nil {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "UNAUTHORIZED"})
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		mfaWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "INVALID_REQUEST"})
		return
	}
	if err := h.mfa.Disable(r.Context(), user.ID, body.Code); err != nil {
		mfaWriteJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MFAChallenge godoc
// @Summary Complete MFA login challenge
// @Tags mfa
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/mfa/challenge [post]
func (h *MFAHandler) MFAChallenge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MFASessionToken string `json:"mfa_session_token"`
		Code            string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MFASessionToken == "" || body.Code == "" {
		mfaWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "INVALID_REQUEST"})
		return
	}
	userID, err := h.mfa.Challenge(r.Context(), body.MFASessionToken, body.Code)
	if err != nil {
		mfaWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	user, err := h.auth.GetUserByID(r.Context(), userID)
	if err != nil {
		mfaWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "USER_NOT_FOUND"})
		return
	}
	refreshToken, err := h.auth.IssueRefreshTokenForUser(r.Context(), userID)
	if err != nil {
		mfaWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "TOKEN_ERROR"})
		return
	}
	accessToken, err := pkgcrypto.GenerateAccessToken(user.ID, user.Email, user.Role, user.IsVerified, h.cfg.JWTSecretKey, h.cfg.JWTAccessExpiryHours)
	if err != nil {
		mfaWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "TOKEN_ERROR"})
		return
	}
	mfaWriteJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// MFALoginResponse is returned by the Login handler when MFA is required.
type MFALoginResponse struct {
	Status          string `json:"status"`
	MFASessionToken string `json:"mfa_session_token"`
}

// MFARequiredFromError unwraps a domain.MFARequiredError if present.
func MFARequiredFromError(err error) (*domain.MFARequiredError, bool) {
	var mfaErr *domain.MFARequiredError
	if errors.As(err, &mfaErr) {
		return mfaErr, true
	}
	return nil, false
}

func mfaWriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
