package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain/contracts"
)

//go:generate mockgen -source=category_service.go -destination=../mocks/category_service.go -package=mocks

// validCategorySortFields are the only column names callers may sort
// category lists by; anything else is rejected rather than silently
// defaulting, so API clients get a clear signal instead of unexpectedly
// sorted results.
var validCategorySortFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
}

type CategoryService interface {
	CreateCategory(ctx context.Context, category *domain.Category) error
	BulkCreateCategories(ctx context.Context, categories []*domain.Category) ([]*domain.Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error)
	// GetCategoryList returns the matching page of categories plus the total
	// count of rows matching filters (ignoring limit/offset), for pagination.
	GetCategoryList(ctx context.Context, filters *domain.CategoryFilters) ([]*domain.Category, int64, error)
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

type categoryService struct {
	categoryRepo contracts.CategoryRepository
	logger       *slog.Logger
}

func NewCategoryService(repo contracts.CategoryRepository, logger *slog.Logger) CategoryService {
	return &categoryService{
		categoryRepo: repo,
		logger:       logger,
	}
}

func (cs *categoryService) BulkCreateCategories(ctx context.Context, categories []*domain.Category) ([]*domain.Category, error) {
	if len(categories) == 0 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "at least one category is required")
	}
	if len(categories) > 500 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "bulk import is limited to 500 categories per request")
	}

	for _, cat := range categories {
		if cat.Name == "" {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "category name is required")
		}
		if cat.Slug == "" {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "category slug is required")
		}
		cat.ID = uuid.New()
	}

	cs.logger.Info("bulk creating categories", "count", len(categories))

	if err := cs.categoryRepo.BulkCreateCategories(ctx, categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func (cs *categoryService) CreateCategory(ctx context.Context, category *domain.Category) error {
	if category.Slug == "" {
		return pkgerrors.NewValidation("INVALID_DATA", "category slug is required")
	}

	if category.Name == "" {
		return pkgerrors.NewValidation("INVALID_DATA", "category name is required")
	}

	category.ID = uuid.New()

	cs.logger.Info("creating category", "slug", category.Slug, "name", category.Name)

	return cs.categoryRepo.CreateCategory(ctx, category)
}

func (cs *categoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	if id == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "category id is required")
	}

	cs.logger.Info("fetching category by id", "id", id)

	return cs.categoryRepo.GetCategoryByID(ctx, id)
}

func (cs *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	if slug == "" {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "category slug is required")
	}

	cs.logger.Info("fetching category by slug", "slug", slug)

	return cs.categoryRepo.GetCategoryBySlug(ctx, slug)
}

func (cs *categoryService) GetCategoryList(ctx context.Context, filters *domain.CategoryFilters) ([]*domain.Category, int64, error) {
	if filters == nil {
		filters = &domain.CategoryFilters{}
	}

	if err := validateSortField(filters.SortBy, validCategorySortFields); err != nil {
		return nil, 0, err
	}

	filters.Limit = capPageSize(filters.Limit, domain.DefaultPageSize, domain.MaxPageSize)

	cs.logger.Info("fetching category list", "limit", filters.Limit, "offset", filters.Offset)

	return cs.categoryRepo.GetCategoryList(ctx, filters)
}

func (cs *categoryService) UpdateCategory(ctx context.Context, category *domain.Category) error {
	if category.ID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "category id is required")
	}

	if category.Name == "" {
		return pkgerrors.NewValidation("INVALID_DATA", "category name is required")
	}

	if category.Slug == "" {
		return pkgerrors.NewValidation("INVALID_DATA", "category slug is required")
	}

	cs.logger.Info("updating category", "id", category.ID, "slug", category.Slug)

	return cs.categoryRepo.UpdateCategory(ctx, category)
}

func (cs *categoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "category id is required")
	}

	cs.logger.Info("deleting category", "id", id)

	return cs.categoryRepo.DeleteCategory(ctx, id)
}
