package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// SKUHandler handles SKU HTTP requests
type SKUHandler struct {
	skuService service.SKUService
}

// NewSKUHandler creates a new SKU handler
func NewSKUHandler(skuService service.SKUService) *SKUHandler {
	return &SKUHandler{skuService}
}

// CreateSKURequest represents the request to create a SKU
type CreateSKURequest struct {
	ProductID    uuid.UUID   `json:"product_id"`
	SKUCode      string      `json:"sku_code"`
	VariantAttrs interface{} `json:"variant_attrs,omitempty"`
	PriceAmount  int64       `json:"price_amount"`
	ComparePrice *int64      `json:"compare_price,omitempty"`
	Currency     string      `json:"currency,omitempty"`
	WeightGrams  *int32      `json:"weight_grams,omitempty"`
	IsActive     bool        `json:"is_active,omitempty"`
}

// UpdateSKURequest represents the request to update a SKU
type UpdateSKURequest struct {
	ProductID    uuid.UUID   `json:"product_id"`
	SKUCode      string      `json:"sku_code"`
	VariantAttrs interface{} `json:"variant_attrs,omitempty"`
	PriceAmount  int64       `json:"price_amount"`
	ComparePrice *int64      `json:"compare_price,omitempty"`
	Currency     string      `json:"currency,omitempty"`
	WeightGrams  *int32      `json:"weight_grams,omitempty"`
	IsActive     bool        `json:"is_active,omitempty"`
}

// CreateSKU creates a new SKU
//
//	@Summary		Create a SKU
//	@Tags			skus
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateSKURequest	true	"SKU payload"
//	@Success		201		{object}	Response{data=domain.SKU}
//	@Failure		400		{object}	Response
//	@Failure		401		{object}	Response
//	@Failure		403		{object}	Response
//	@Failure		409		{object}	Response
//	@Router			/api/v1/skus [post]
func (h *SKUHandler) CreateSKU(w http.ResponseWriter, r *http.Request) {
	var req CreateSKURequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	sku := &domain.SKU{
		ProductID:    req.ProductID,
		SKUCode:      req.SKUCode,
		PriceAmount:  req.PriceAmount,
		ComparePrice: req.ComparePrice,
		Currency:     req.Currency,
		WeightGrams:  req.WeightGrams,
		IsActive:     req.IsActive,
	}

	if err := h.skuService.CreateSKU(r.Context(), sku); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, sku)
}

// GetSKUByID returns a SKU by ID
//
//	@Summary		Get SKU by ID
//	@Tags			skus
//	@Produce		json
//	@Param			id	path		string	true	"SKU UUID"
//	@Success		200	{object}	Response{data=domain.SKU}
//	@Failure		400	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/skus/{id} [get]
func (h *SKUHandler) GetSKUByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid sku id")
		return
	}

	sku, err := h.skuService.GetSKUByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, sku)
}

// GetSKUList returns a list of SKUs
//
//	@Summary		List SKUs
//	@Tags			skus
//	@Produce		json
//	@Param			limit		query		int		false	"Page size (default 20)"
//	@Param			offset		query		int		false	"Page offset (default 0)"
//	@Param			product_id	query		string	false	"Filter by product UUID"
//	@Param			sku_code	query		string	false	"Filter by SKU code (partial match)"
//	@Param			is_active	query		bool	false	"Filter by active status"
//	@Success		200			{object}	Response{data=[]domain.SKU}
//	@Router			/api/v1/skus [get]
func (h *SKUHandler) GetSKUList(w http.ResponseWriter, r *http.Request) {
	limit, offset := GetLimitOffset(r, 20, 0)

	filters := &domain.SKUFilters{
		Limit:  limit,
		Offset: offset,
	}

	// Optional filters from query params
	if productIDStr := r.URL.Query().Get("product_id"); productIDStr != "" {
		if productID, err := uuid.Parse(productIDStr); err == nil {
			filters.ProductID = &productID
		}
	}

	if skuCode := r.URL.Query().Get("sku_code"); skuCode != "" {
		filters.SKUCode = &skuCode
	}

	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filters.IsActive = &isActive
	}

	skus, err := h.skuService.GetSKUList(r.Context(), filters)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, skus)
}

// UpdateSKU updates a SKU
//
//	@Summary		Update SKU
//	@Tags			skus
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string			true	"SKU UUID"
//	@Param			body	body		UpdateSKURequest	true	"SKU payload"
//	@Success		200		{object}	Response{data=domain.SKU}
//	@Failure		400		{object}	Response
//	@Failure		401		{object}	Response
//	@Failure		403		{object}	Response
//	@Failure		404		{object}	Response
//	@Router			/api/v1/skus/{id} [put]
func (h *SKUHandler) UpdateSKU(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid sku id")
		return
	}

	var req UpdateSKURequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	sku := &domain.SKU{
		ID:           id,
		ProductID:    req.ProductID,
		SKUCode:      req.SKUCode,
		PriceAmount:  req.PriceAmount,
		ComparePrice: req.ComparePrice,
		Currency:     req.Currency,
		WeightGrams:  req.WeightGrams,
		IsActive:     req.IsActive,
	}

	if err := h.skuService.UpdateSKU(r.Context(), sku); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, sku)
}

// DeleteSKU deletes a SKU
//
//	@Summary		Delete SKU
//	@Tags			skus
//	@Security		BearerAuth
//	@Param			id	path	string	true	"SKU UUID"
//	@Success		204
//	@Failure		400	{object}	Response
//	@Failure		401	{object}	Response
//	@Failure		403	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/skus/{id} [delete]
func (h *SKUHandler) DeleteSKU(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid sku id")
		return
	}

	if err := h.skuService.DeleteSKU(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusNoContent, nil)
}
