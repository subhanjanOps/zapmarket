package http

import (
	"encoding/json"
	"net/http"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

// SendOTPRequest is the body for POST /v1/auth/otp/send
type SendOTPRequest struct {
	Email string `json:"email"`
}

// VerifyOTPRequest is the body for POST /v1/auth/otp/verify
type VerifyOTPRequest struct {
	Code string `json:"code"`
}

// ForgotPasswordOTPRequest is the body for POST /v1/auth/password/forgot-otp
type ForgotPasswordOTPRequest struct {
	Phone string `json:"phone"`
}

// ResetPasswordOTPRequest is the body for POST /v1/auth/password/reset-otp
type ResetPasswordOTPRequest struct {
	Phone       string `json:"phone"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// SendOTP handles POST /v1/auth/otp/send
//
// @Summary Resend email OTP
// @Description Issues a new email OTP to the given address. Use after registration if the original OTP was lost.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body SendOTPRequest true "Email"
// @Success 200 {object} map[string]string
// @Router /otp/send [post]
func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		h.writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	_ = h.authSvc.SendEmailOTP(r.Context(), req.Email)
	h.writeResponse(w, http.StatusOK, map[string]string{
		"message": "if an account with that email exists, a new OTP has been sent",
	})
}

// VerifyOTP handles POST /v1/auth/otp/verify
// Requires a valid JWT in the Authorization header to identify the user.
//
// @Summary Verify email OTP
// @Description Verifies the submitted OTP and marks the account as verified.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body VerifyOTPRequest true "OTP code"
// @Success 200 {object} map[string]string
// @Failure 401 {object} AuthResponse "Invalid or expired OTP"
// @Router /otp/verify [post]
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Code == "" {
		h.writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	// Extract user from JWT.
	tokenStr, ok := bearerToken(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	user, err := h.authSvc.ValidateAccessToken(r.Context(), tokenStr)
	if err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}

	if err := h.authSvc.VerifyEmailOTP(r.Context(), user.ID, req.Code); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"message": "email verified successfully"})
}

// ForgotPasswordOTP handles POST /v1/auth/password/forgot-otp
//
// @Summary Request a password reset via phone OTP
// @Description Sends an OTP to the phone number if an account with that number exists. Always returns 200.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ForgotPasswordOTPRequest true "Phone number"
// @Success 200 {object} map[string]string
// @Router /password/forgot-otp [post]
func (h *Handler) ForgotPasswordOTP(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Phone == "" {
		h.writeError(w, http.StatusBadRequest, "phone is required")
		return
	}
	_ = h.authSvc.RequestPasswordResetOTP(r.Context(), req.Phone)
	h.writeResponse(w, http.StatusOK, map[string]string{
		"message": "if an account with that phone number exists, an OTP has been sent",
	})
}

// ResetPasswordOTP handles POST /v1/auth/password/reset-otp
//
// @Summary Reset password using a phone OTP
// @Description Verifies the OTP and updates the user's password.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetPasswordOTPRequest true "Phone, OTP code, and new password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} AuthResponse
// @Failure 401 {object} AuthResponse
// @Router /password/reset-otp [post]
func (h *Handler) ResetPasswordOTP(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Phone == "" || req.Code == "" || req.NewPassword == "" {
		h.writeError(w, http.StatusBadRequest, "phone, code, and new_password are required")
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

	if err := h.authSvc.ResetPasswordWithOTP(r.Context(), req.Phone, req.Code, req.NewPassword); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

