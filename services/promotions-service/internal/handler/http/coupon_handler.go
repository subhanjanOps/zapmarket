package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	domainerrors "github.com/zapmarket/zapmarket/services/promotions-service/internal/domain/errors"
	"github.com/zapmarket/zapmarket/services/promotions-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/promotions-service/internal/domain"
)

type couponRedeemer interface {
	RecordUsage(ctx context.Context, couponID, userID, orderID string) error
	GetActiveSales(ctx context.Context) ([]*domain.Coupon, error)
}

type CouponHandler struct {
	validate *usecases.ValidateCouponUseCase
	repo     couponRedeemer
}

func NewCouponHandler(validate *usecases.ValidateCouponUseCase, repo couponRedeemer) *CouponHandler {
	return &CouponHandler{validate: validate, repo: repo}
}

type validateRequest struct {
	Code           string `json:"code"`
	UserID         string `json:"user_id"`
	CartTotalPaise int64  `json:"cart_total_paise"`
}

type validateResponse struct {
	CouponID      string `json:"coupon_id"`
	DiscountPaise int64  `json:"discount_paise"`
	FinalPaise    int64  `json:"final_paise"`
}

func (h *CouponHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	result, err := h.validate.Execute(r.Context(), usecases.ValidateCouponInput{
		Code:           req.Code,
		UserID:         req.UserID,
		CartTotalPaise: req.CartTotalPaise,
	})
	if err != nil {
		if errors.Is(err, domainerrors.ErrCouponNotFound) {
			writeError(w, http.StatusNotFound, "COUPON_NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "COUPON_INVALID", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, validateResponse{
		CouponID:      result.CouponID,
		DiscountPaise: result.DiscountPaise,
		FinalPaise:    result.FinalPaise,
	})
}

type redeemRequest struct {
	UserID  string `json:"user_id"`
	OrderID string `json:"order_id"`
	// CouponID is embedded in the URL path as {coupon_id}
}

func (h *CouponHandler) Redeem(w http.ResponseWriter, r *http.Request) {
	couponID := r.PathValue("coupon_id")
	if couponID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_COUPON_ID", "coupon_id path param required")
		return
	}
	var req redeemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.repo.RecordUsage(r.Context(), couponID, req.UserID, req.OrderID); err != nil {
		writeError(w, http.StatusInternalServerError, "REDEEM_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CouponHandler) GetActiveSales(w http.ResponseWriter, r *http.Request) {
	sales, err := h.repo.GetActiveSales(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sales)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"data": v})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
