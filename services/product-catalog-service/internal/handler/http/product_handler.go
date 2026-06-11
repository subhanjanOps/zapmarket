package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// ProductHandler handles product HTTP requests
type ProductHandler struct {
	productService service.ProductService
}

// NewProductHandler creates a new product handler
func NewProductHandler(productService service.ProductService) *ProductHandler {
	return &ProductHandler{productService}
}

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	CategoryID  uuid.UUID   `json:"category_id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description *string     `json:"description,omitempty"`
	Attributes  interface{} `json:"attributes,omitempty"`
	Status      string      `json:"status,omitempty"`
}

// UpdateProductRequest represents the request to update a product
type UpdateProductRequest struct {
	CategoryID  uuid.UUID   `json:"category_id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description *string     `json:"description,omitempty"`
	Attributes  interface{} `json:"attributes,omitempty"`
	Status      string      `json:"status,omitempty"`
}

// CreateProduct creates a new product
//
//	@Summary		Create a product
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateProductRequest	true	"Product payload"
//	@Success		201		{object}	Response{data=domain.Product}
//	@Failure		400		{object}	Response
//	@Failure		401		{object}	Response
//	@Failure		403		{object}	Response
//	@Failure		409		{object}	Response
//	@Router			/api/v1/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	// In a real app, seller_id would come from the authenticated user context
	// For now, we'll require it in the request or use a default
	sellerID := uuid.New() // This should come from auth context in production

	product := &domain.Product{
		CategoryID:  req.CategoryID,
		SellerID:    sellerID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      domain.ProductStatus(req.Status),
	}

	// Handle attributes convertion if provided
	if req.Attributes != nil {
		// In a real app, you would marshal this to json.RawMessage
		// For now, we'll initialize as empty
		product.Attributes = nil
	}

	if err := h.productService.CreateProduct(r.Context(), product); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, product)
}

// GetProductByID returns a product by ID
//
//	@Summary		Get product by ID
//	@Tags			products
//	@Produce		json
//	@Param			id	path		string	true	"Product UUID"
//	@Success		200	{object}	Response{data=domain.Product}
//	@Failure		400	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/products/{id} [get]
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid product id")
		return
	}

	product, err := h.productService.GetProductByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, product)
}

// GetProductBySlug returns a product by slug
//
//	@Summary		Get product by slug
//	@Tags			products
//	@Produce		json
//	@Param			slug	path		string	true	"Product slug"
//	@Success		200		{object}	Response{data=domain.Product}
//	@Failure		400		{object}	Response
//	@Failure		404		{object}	Response
//	@Router			/api/v1/products/slug/{slug} [get]
func (h *ProductHandler) GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_SLUG", "slug is required")
		return
	}

	product, err := h.productService.GetProductBySlug(r.Context(), slug)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, product)
}

// GetProductList returns a list of products
//
//	@Summary		List products
//	@Tags			products
//	@Produce		json
//	@Param			limit		query		int		false	"Page size (default 20)"
//	@Param			offset		query		int		false	"Page offset (default 0)"
//	@Param			category_id	query		string	false	"Filter by category UUID"
//	@Param			status		query		string	false	"Filter by status (draft|active|inactive|archived)"
//	@Param			search		query		string	false	"Search by name or description"
//	@Success		200			{object}	Response{data=[]domain.Product}
//	@Router			/api/v1/products [get]
func (h *ProductHandler) GetProductList(w http.ResponseWriter, r *http.Request) {
	limit, offset := GetLimitOffset(r, 20, 0)

	filters := &domain.ProductFilters{
		Limit:  limit,
		Offset: offset,
	}

	// Optional filters from query params
	if categoryIDStr := r.URL.Query().Get("category_id"); categoryIDStr != "" {
		if categoryID, err := uuid.Parse(categoryIDStr); err == nil {
			filters.CategoryID = &categoryID
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = search
	}

	products, err := h.productService.GetProductList(r.Context(), filters)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, products)
}

// UpdateProduct updates a product
//
//	@Summary		Update product
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Product UUID"
//	@Param			body	body		UpdateProductRequest	true	"Product payload"
//	@Success		200		{object}	Response{data=domain.Product}
//	@Failure		400		{object}	Response
//	@Failure		401		{object}	Response
//	@Failure		403		{object}	Response
//	@Failure		404		{object}	Response
//	@Router			/api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid product id")
		return
	}

	var req UpdateProductRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	// Fetch existing product to preserve seller_id
	existingProduct, err := h.productService.GetProductByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	product := &domain.Product{
		ID:          id,
		CategoryID:  req.CategoryID,
		SellerID:    existingProduct.SellerID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      domain.ProductStatus(req.Status),
	}

	if err := h.productService.UpdateProduct(r.Context(), product); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, product)
}

// DeleteProduct deletes a product
//
//	@Summary		Delete product
//	@Tags			products
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Product UUID"
//	@Success		204
//	@Failure		400	{object}	Response
//	@Failure		401	{object}	Response
//	@Failure		403	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid product id")
		return
	}

	if err := h.productService.DeleteProduct(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusNoContent, nil)
}
