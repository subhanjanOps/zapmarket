package http

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

// maxImageUploadBytes caps the request body read for an image upload,
// rejecting oversized files before they're fully buffered in memory.
const maxImageUploadBytes = 5 << 20 // 5 MiB

// CreateProductImage uploads a new product image to object storage and
// records it.
//
//	@Summary		Upload a product image
//	@Tags			product-images
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product_id	path		string	true	"Product UUID"
//	@Param			sku_id		formData	string	false	"SKU UUID (optional)"
//	@Param			file		formData	file	true	"Image file (png, jpeg, webp, or gif; max 5MB)"
//	@Success		201			{object}	Response{data=domain.ProductImage}
//	@Failure		400			{object}	Response
//	@Failure		401			{object}	Response
//	@Failure		403			{object}	Response
//	@Router			/v1/products/{product_id}/images [post]
func (h *ProductImageHandler) CreateProductImage(w http.ResponseWriter, r *http.Request) {
	productIDStr := chi.URLParam(r, "product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid product id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageUploadBytes)
	if err := r.ParseMultipartForm(maxImageUploadBytes); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "FILE_TOO_LARGE", "file exceeds the 5MB upload limit")
		return
	}

	var skuID *uuid.UUID
	if skuIDStr := r.FormValue("sku_id"); skuIDStr != "" {
		parsed, err := uuid.Parse(skuIDStr)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_SKU_ID", "invalid sku_id")
			return
		}
		skuID = &parsed
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "MISSING_FILE", "file is required")
		return
	}
	defer file.Close()

	// Sniff the real content type from the file bytes rather than trusting
	// a client-supplied header, which is easy to spoof and tells us nothing
	// about what's actually in the body.
	sniffBuf := make([]byte, 512)
	n, err := file.Read(sniffBuf)
	if err != nil && err != io.EOF {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_FILE", "could not read uploaded file")
		return
	}
	contentType := http.DetectContentType(sniffBuf[:n])

	// Reassemble a reader over the sniffed bytes + the rest of the file,
	// since the sniff read already consumed the first 512 bytes.
	fullReader := io.MultiReader(bytes.NewReader(sniffBuf[:n]), file)

	size, err := formFileSize(r, "file")
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_FILE", "could not determine file size")
		return
	}

	image, err := h.imageService.UploadProductImage(r.Context(), service.UploadProductImageInput{
		ProductID:   productID,
		SKUID:       skuID,
		Reader:      fullReader,
		Size:        size,
		ContentType: contentType,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, image)
}

// formFileSize reads the Size of the named multipart file part without
// consuming its reader, by going through the parsed multipart form header
// rather than the (already partially-read) file handle.
func formFileSize(r *http.Request, field string) (int64, error) {
	if r.MultipartForm == nil || r.MultipartForm.File[field] == nil || len(r.MultipartForm.File[field]) == 0 {
		return 0, http.ErrMissingFile
	}
	return r.MultipartForm.File[field][0].Size, nil
}

// GetImagesByProductID returns images by product ID
//
//	@Summary		Get images by product
//	@Tags			product-images
//	@Produce		json
//	@Param			product_id	path		string	true	"Product UUID"
//	@Success		200			{object}	Response{data=[]domain.ProductImage}
//	@Failure		400			{object}	Response
//	@Router			/v1/products/{product_id}/images [get]
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
//	@Router			/v1/products/{product_id}/images/sku/{sku_id} [get]
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
//	@Router			/v1/products/{product_id}/images/{id}/position [patch]
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
//	@Router			/v1/products/{product_id}/images/{id} [delete]
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
