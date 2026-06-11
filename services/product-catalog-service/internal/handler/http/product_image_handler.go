package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// ProductImageHandler handles product image HTTP requests
type ProductImageHandler struct {
	imageService service.ProductImageService
}

// NewProductImageHandler creates a new product image handler
func NewProductImageHandler(imageService service.ProductImageService) *ProductImageHandler {
	return &ProductImageHandler{imageService}
}

// CreateProductImageRequest represents the request to create a product image
type CreateProductImageRequest struct {
	ProductID uuid.UUID  `json:"product_id"`
	SKUID     *uuid.UUID `json:"sku_id,omitempty"`
	URL       string     `json:"url"`
}

// CreateProductImage creates a new product image
//
//	@Summary		Add product image
//	@Tags			product-images
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product_id	path		string						true	"Product UUID"
//	@Param			body		body		CreateProductImageRequest	true	"Image payload"
//	@Success		201			{object}	Response{data=domain.ProductImage}
//	@Failure		400			{object}	Response
//	@Failure		401			{object}	Response
//	@Failure		403			{object}	Response
//	@Router			/api/v1/products/{product_id}/images [post]
func (h *ProductImageHandler) CreateProductImage(w http.ResponseWriter, r *http.Request) {
	var req CreateProductImageRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	image := &domain.ProductImage{
		ProductID: req.ProductID,
		SKUId:     req.SKUID,
		URL:       req.URL,
	}

	if err := h.imageService.CreateProductImage(r.Context(), image); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, image)
}

// GetImagesByProductID returns images by product ID
//
//	@Summary		Get images by product
//	@Tags			product-images
//	@Produce		json
//	@Param			product_id	path		string	true	"Product UUID"
//	@Success		200			{object}	Response{data=[]domain.ProductImage}
//	@Failure		400			{object}	Response
//	@Router			/api/v1/products/{product_id}/images [get]
func (h *ProductImageHandler) GetImagesByProductID(w http.ResponseWriter, r *http.Request) {
	productIDStr := chi.URLParam(r, "product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid product id")
		return
	}

	images, err := h.imageService.GetImagesByProductID(r.Context(), productID)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, images)
}

// GetImagesBySKUID returns images by SKU ID
//
//	@Summary		Get images by SKU
//	@Tags			product-images
//	@Produce		json
//	@Param			product_id	path		string	true	"Product UUID"
//	@Param			sku_id		path		string	true	"SKU UUID"
//	@Success		200			{object}	Response{data=[]domain.ProductImage}
//	@Failure		400			{object}	Response
//	@Router			/api/v1/products/{product_id}/images/sku/{sku_id} [get]
func (h *ProductImageHandler) GetImagesBySKUID(w http.ResponseWriter, r *http.Request) {
	skuIDStr := chi.URLParam(r, "sku_id")
	skuID, err := uuid.Parse(skuIDStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid sku id")
		return
	}

	images, err := h.imageService.GetImagesBySKUID(r.Context(), skuID)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, images)
}

// UpdateImagePositionRequest represents the request to update image position
type UpdateImagePositionRequest struct {
	Position int `json:"position"`
}

// UpdateImagePosition updates the position of a product image
//
//	@Summary		Update image position
//	@Tags			product-images
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product_id	path		string						true	"Product UUID"
//	@Param			id			path		string						true	"Image UUID"
//	@Param			body		body		UpdateImagePositionRequest	true	"Position payload"
//	@Success		200			{object}	Response
//	@Failure		400			{object}	Response
//	@Failure		401			{object}	Response
//	@Failure		403			{object}	Response
//	@Router			/api/v1/products/{product_id}/images/{id}/position [patch]
func (h *ProductImageHandler) UpdateImagePosition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid image id")
		return
	}

	var req UpdateImagePositionRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.imageService.UpdateImagePosition(r.Context(), id, req.Position); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, nil)
}

// DeleteProductImage deletes a product image
//
//	@Summary		Delete product image
//	@Tags			product-images
//	@Security		BearerAuth
//	@Param			product_id	path	string	true	"Product UUID"
//	@Param			id			path	string	true	"Image UUID"
//	@Success		204
//	@Failure		400	{object}	Response
//	@Failure		401	{object}	Response
//	@Failure		403	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/products/{product_id}/images/{id} [delete]
func (h *ProductImageHandler) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid image id")
		return
	}

	if err := h.imageService.DeleteProductImage(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusNoContent, nil)
}
