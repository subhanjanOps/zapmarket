package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// SendPhoneOTP handles POST /v1/auth/phone-otp/send
func (h *Handler) SendPhoneOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Phone  string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	if req.Phone == "" {
		h.writeError(w, http.StatusBadRequest, "phone is required")
		return
	}
	if err := h.authSvc.SendPhoneOTP(r.Context(), userID, req.Phone); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"message": "OTP sent"})
}

// VerifyPhoneOTP handles POST /v1/auth/phone-otp/verify
func (h *Handler) VerifyPhoneOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	if err := h.authSvc.VerifyPhoneOTP(r.Context(), userID, req.Code); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"message": "phone verified"})
}

// UpdateRegistrationProfile handles PATCH /v1/auth/registration/profile
func (h *Handler) UpdateRegistrationProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       string  `json:"user_id"`
		DOB          string  `json:"dob"`
		Gender       *string `json:"gender"`
		AddressLine1 string  `json:"address_line1"`
		AddressLine2 *string `json:"address_line2"`
		City         string  `json:"city"`
		State        string  `json:"state"`
		Country      string  `json:"country"`
		Pincode      string  `json:"pincode"`
		StoreName    *string `json:"store_name"`
		Tagline      *string `json:"tagline"`
		Category     *string `json:"category"`
		BusinessPhone *string `json:"business_phone"`
		BusinessType  *string `json:"business_type"`
		TaxID        *string `json:"tax_id"`
		BizLine1     *string `json:"biz_line1"`
		BizCity      *string `json:"biz_city"`
		BizState     *string `json:"biz_state"`
		BizCountry   *string `json:"biz_country"`
		BizPincode   *string `json:"biz_pincode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	dob, err := time.Parse("2006-01-02", req.DOB)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "dob must be in YYYY-MM-DD format")
		return
	}
	if req.Country == "" {
		req.Country = "IN"
	}
	if req.AddressLine1 == "" || req.City == "" || req.State == "" || req.Pincode == "" {
		h.writeError(w, http.StatusBadRequest, "address_line1, city, state, and pincode are required")
		return
	}

	addr := domain.Address{
		Line1:   req.AddressLine1,
		Line2:   req.AddressLine2,
		City:    req.City,
		State:   req.State,
		Country: req.Country,
		Pincode: req.Pincode,
	}

	var sp *domain.SellerProfile
	if req.StoreName != nil {
		bType := "individual"
		if req.BusinessType != nil {
			bType = *req.BusinessType
		}
		tagline := ""
		if req.Tagline != nil {
			tagline = *req.Tagline
		}
		category := ""
		if req.Category != nil {
			category = *req.Category
		}
		bPhone := ""
		if req.BusinessPhone != nil {
			bPhone = *req.BusinessPhone
		}
		sp = &domain.SellerProfile{
			StoreName:    *req.StoreName,
			Tagline:      tagline,
			Category:     category,
			BusinessPhone: bPhone,
			City:         req.City,
			Pincode:      req.Pincode,
			BusinessType: bType,
			TaxID:        req.TaxID,
			BizLine1:     req.BizLine1,
			BizCity:      req.BizCity,
			BizState:     req.BizState,
			BizCountry:   req.BizCountry,
			BizPincode:   req.BizPincode,
		}
	}

	if err := h.authSvc.UpdateRegistrationProfile(r.Context(), userID, dob, req.Gender, nil, addr, sp); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"message": "profile saved"})
}

// CompleteRegistration handles POST /v1/auth/registration/complete
func (h *Handler) CompleteRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID        string `json:"user_id"`
		TermsAccepted bool   `json:"terms_accepted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !req.TermsAccepted {
		h.writeError(w, http.StatusBadRequest, "terms of service must be accepted")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	if err := h.authSvc.CompleteRegistration(r.Context(), userID); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"message": "registration complete"})
}

// UploadProfilePicture handles POST /v1/auth/profile/picture (multipart/form-data, field: picture, form: user_id)
func (h *Handler) UploadProfilePicture(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.FormValue("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		h.writeError(w, http.StatusBadRequest, "file too large (max 5 MB)")
		return
	}

	file, header, err := r.FormFile("picture")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "picture field is required")
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")
	if ct != "image/jpeg" && ct != "image/png" {
		h.writeError(w, http.StatusBadRequest, "only JPEG and PNG images are accepted")
		return
	}
	ext := ".jpg"
	if ct == "image/png" {
		ext = ".png"
	}

	if err := os.MkdirAll(h.pfpDir, 0755); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to prepare upload directory")
		return
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	dst := filepath.Join(h.pfpDir, filename)
	out, err := os.Create(dst)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	pfpURL := strings.TrimRight(h.pfpBaseURL, "/") + "/" + filename
	zeroTime := time.Time{}
	if err := h.authSvc.UpdateRegistrationProfile(r.Context(), userID, zeroTime, nil, &pfpURL, domain.Address{}, nil); err != nil {
		pkgerrors.HandleHTTP(w, err)
		return
	}
	h.writeResponse(w, http.StatusOK, map[string]string{"pfp_url": pfpURL})
}
