package http

import (
	"encoding/json"
	"net/http"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

// ForgotPasswordRequest is the body for POST /v1/auth/password/forgot
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the body for POST /v1/auth/password/reset
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ForgotPassword handles POST /v1/auth/password/forgot
// Always returns 200 to avoid leaking whether the email is registered.
//
// @Summary Request a password reset
// @Description Sends a password reset email if the address is registered. Always returns 200.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ForgotPasswordRequest true "Email address"
// @Success 200 {object} map[string]string "Reset email sent"
// @Router /password/forgot [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		h.writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Intentionally ignore the error — we never reveal whether an email exists.
	_ = h.authSvc.RequestPasswordReset(r.Context(), req.Email)

	h.writeResponse(w, http.StatusOK, map[string]string{
		"message": "if an account with that email exists, a reset link has been sent",
	})
}

// ResetPassword handles POST /v1/auth/password/reset
//
// @Summary Reset password using a reset token
// @Description Validates the token from the reset email and updates the user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} map[string]string "Password updated"
// @Failure 400 {object} AuthResponse "Missing fields or invalid password"
// @Failure 401 {object} AuthResponse "Invalid or expired token"
// @Router /password/reset [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" || req.NewPassword == "" {
		h.writeError(w, http.StatusBadRequest, "token and new_password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		h.writeError(w, http.StatusBadRequest, "new_password must be at least 8 characters")
		return
	}
	if len(req.NewPassword) > 72 {
		h.writeError(w, http.StatusBadRequest, "new_password must be 72 characters or fewer")
		return
	}

	if err := h.authSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{
		"message": "password updated successfully",
	})
}
