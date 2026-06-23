package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/authctx"
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
//	@Router			/v1/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	user := authctx.UserFromContext(r.Context())
	if user == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	sellerID, err := uuid.Parse(user.Id)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INVALID_USER_ID", "authenticated user id is not a valid UUID")
		return
	}

	attrs, err := attributesToRawMessage(req.Attributes)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ATTRIBUTES", "attributes must be valid JSON")
		return
	}

	product := &domain.Product{
		CategoryID:  req.CategoryID,
		SellerID:    sellerID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Attributes:  attrs,
		Status:      domain.ProductStatus(req.Status),
	}

	if err := h.productService.CreateProduct(r.Context(), product); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, product)
}

// attributesToRawMessage marshals a decoded JSON value (or nil) back into
// json.RawMessage. The products.attributes column is NOT NULL with a '{}'
// default at the schema level, but that default only applies when the
// column is omitted from an INSERT/UPDATE entirely — since the repository
// always supplies an explicit value, a nil here would insert SQL NULL and
// violate the constraint. Defaulting nil to "{}" up front avoids that.
func attributesToRawMessage(v interface{}) (json.RawMessage, error) {
	if v == nil {
		return json.RawMessage("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
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
//	@Router			/v1/products/{id} [get]
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
//	@Router			/v1/products/slug/{slug} [get]
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

// GetProductList returns a paginated list of products
//
//	@Summary		List products
//	@Tags			products
//	@Produce		json
//	@Param			limit		query		int		false	"Page size (default 20, max 100)"
//	@Param			offset		query		int		false	"Page offset (default 0)"
//	@Param			category_id	query		string	false	"Filter by category UUID"
//	@Param			seller_id	query		string	false	"Filter by seller UUID"
//	@Param			status		query		string	false	"Filter by status (draft|active|inactive|archived)"
//	@Param			search		query		string	false	"Search by name or description"
//	@Param			sort_by		query		string	false	"Sort field: name|created_at|updated_at"
//	@Param			sort_order	query		string	false	"Sort direction: asc|desc"
//	@Success		200			{object}	Response{data=[]domain.Product}
//	@Failure		400			{object}	Response
//	@Router			/v1/products [get]
func (h *ProductHandler) GetProductList(w http.ResponseWriter, r *http.Request) {
	limit, offset := GetLimitOffset(r, domain.DefaultPageSize, 0)

	filters := &domain.ProductFilters{
		Status:    r.URL.Query().Get("status"),
		Search:    r.URL.Query().Get("search"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
		Limit:     limit,
		Offset:    offset,
	}

	if categoryIDStr := r.URL.Query().Get("category_id"); categoryIDStr != "" {
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "invalid category_id")
			return
		}
		filters.CategoryID = &categoryID
	}

	if sellerIDStr := r.URL.Query().Get("seller_id"); sellerIDStr != "" {
		sellerID, err := uuid.Parse(sellerIDStr)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_SELLER_ID", "invalid seller_id")
			return
		}
		filters.SellerID = &sellerID
	}

	// Sellers can only see their own products regardless of any query param.
	if user := authctx.UserFromContext(r.Context()); user != nil && user.Role == "seller" {
		sellerID, err := uuid.Parse(user.Id)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "INVALID_USER_ID", "authenticated user id is not a valid UUID")
			return
		}
		filters.SellerID = &sellerID
	}

	products, total, err := h.productService.GetProductList(r.Context(), filters)
	if err != nil {
		HandleError(w, err)
		return
	}

	page := 1
	if filters.Limit > 0 {
		page = filters.Offset/filters.Limit + 1
	}
	httpx.Paginated(w, http.StatusOK, products, total, page, filters.Limit)
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
//	@Router			/v1/products/{id} [put]
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

	user, err := requireUser(r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if err := assertOwnership(r.Context(), h.productService, id, user); err != nil {
		HandleError(w, err)
		return
	}

	// Fetch existing product to preserve seller_id
	existingProduct, err := h.productService.GetProductByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Merge: start from existing, apply only the fields the caller provided.
	// UpdatedAt is preserved so the repository can use it for optimistic
	// locking (WHERE updated_at = $8).
	product := &domain.Product{
		ID:          id,
		CategoryID:  existingProduct.CategoryID,
		SellerID:    existingProduct.SellerID,
		Name:        existingProduct.Name,
		Slug:        existingProduct.Slug,
		Description: existingProduct.Description,
		Attributes:  existingProduct.Attributes,
		Status:      existingProduct.Status,
		UpdatedAt:   existingProduct.UpdatedAt,
	}
	if req.CategoryID != uuid.Nil {
		product.CategoryID = req.CategoryID
	}
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Slug != "" {
		product.Slug = req.Slug
	}
	if req.Description != nil {
		product.Description = req.Description
	}
	if req.Status != "" {
		product.Status = domain.ProductStatus(req.Status)
	}
	if req.Attributes != nil {
		attrs, err := attributesToRawMessage(req.Attributes)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_ATTRIBUTES", "attributes must be valid JSON")
			return
		}
		product.Attributes = attrs
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
//	@Router			/v1/products/{id} [delete]
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
