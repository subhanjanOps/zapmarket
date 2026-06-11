package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// CategoryHandler handles category HTTP requests
type CategoryHandler struct {
	categoryService service.CategoryService
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService}
}

// CreateCategoryRequest represents the request to create a category
type CreateCategoryRequest struct {
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

// UpdateCategoryRequest represents the request to update a category
type UpdateCategoryRequest struct {
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

// CreateCategory creates a new category
//
//	@Summary		Create a category
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateCategoryRequest	true	"Category payload"
//	@Success		201		{object}	Response{data=domain.Category}
//	@Failure		400		{object}	Response
//	@Failure		409		{object}	Response
//	@Router			/api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	category := &domain.Category{
		Name:     req.Name,
		Slug:     req.Slug,
		ParentID: req.ParentID,
	}

	if err := h.categoryService.CreateCategory(r.Context(), category); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusCreated, category)
}

// GetCategoryByID returns a category by ID
//
//	@Summary		Get category by ID
//	@Tags			categories
//	@Produce		json
//	@Param			id	path		string	true	"Category UUID"
//	@Success		200	{object}	Response{data=domain.Category}
//	@Failure		400	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/categories/{id} [get]
func (h *CategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid category id")
		return
	}

	category, err := h.categoryService.GetCategoryByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, category)
}

// GetCategoryBySlug returns a category by slug
//
//	@Summary		Get category by slug
//	@Tags			categories
//	@Produce		json
//	@Param			slug	path		string	true	"Category slug"
//	@Success		200		{object}	Response{data=domain.Category}
//	@Failure		400		{object}	Response
//	@Failure		404		{object}	Response
//	@Router			/api/v1/categories/slug/{slug} [get]
func (h *CategoryHandler) GetCategoryBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_SLUG", "slug is required")
		return
	}

	category, err := h.categoryService.GetCategoryBySlug(r.Context(), slug)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, category)
}

// GetCategoryList returns a list of categories
//
//	@Summary		List categories
//	@Tags			categories
//	@Produce		json
//	@Param			limit	query		int		false	"Page size (default 20)"
//	@Param			offset	query		int		false	"Page offset (default 0)"
//	@Success		200		{object}	Response{data=[]domain.Category}
//	@Router			/api/v1/categories [get]
func (h *CategoryHandler) GetCategoryList(w http.ResponseWriter, r *http.Request) {
	limit, offset := GetLimitOffset(r, 20, 0)
	filters := map[string]string{}

	categories, err := h.categoryService.GetCategoryList(r.Context(), filters, limit, offset)
	if err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, categories)
}

// UpdateCategory updates a category
//
//	@Summary		Update category
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Category UUID"
//	@Param			body	body		UpdateCategoryRequest	true	"Category payload"
//	@Success		200		{object}	Response{data=domain.Category}
//	@Failure		400		{object}	Response
//	@Failure		404		{object}	Response
//	@Router			/api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid category id")
		return
	}

	var req UpdateCategoryRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	category := &domain.Category{
		ID:       id,
		Name:     req.Name,
		Slug:     req.Slug,
		ParentID: req.ParentID,
	}

	if err := h.categoryService.UpdateCategory(r.Context(), category); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusOK, category)
}

// DeleteCategory deletes a category
//
//	@Summary		Delete category
//	@Tags			categories
//	@Param			id	path	string	true	"Category UUID"
//	@Success		204
//	@Failure		400	{object}	Response
//	@Failure		404	{object}	Response
//	@Router			/api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "invalid category id")
		return
	}

	if err := h.categoryService.DeleteCategory(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	SuccessResponse(w, http.StatusNoContent, nil)
}
