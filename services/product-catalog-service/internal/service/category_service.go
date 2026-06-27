package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain/contracts"
)

func uniqueStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := ss[:0]
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

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
	BulkCreateCategories(ctx context.Context, inputs []*domain.BulkCategoryInput) ([]*domain.Category, error)
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

func (cs *categoryService) BulkCreateCategories(ctx context.Context, inputs []*domain.BulkCategoryInput) ([]*domain.Category, error) {
	if len(inputs) == 0 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "at least one category is required")
	}
	if len(inputs) > 500 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "bulk import is limited to 500 categories per request")
	}

	for _, in := range inputs {
		if in.Name == "" {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "category name is required")
		}
		if in.Slug == "" {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "category slug is required")
		}
	}

	// Index every input by both name and slug so parent references work
	// regardless of which convention the CSV uses.
	inBatch := make(map[string]struct{}, len(inputs)*2)
	for _, in := range inputs {
		inBatch[in.Name] = struct{}{}
		inBatch[in.Slug] = struct{}{}
	}

	// Collect parent references that must be resolved from the DB — only
	// those not satisfiable within this same batch.
	var externalRefs []string
	for _, in := range inputs {
		if in.ParentName != nil && *in.ParentName != "" {
			if _, ok := inBatch[*in.ParentName]; !ok {
				externalRefs = append(externalRefs, *in.ParentName)
			}
		}
	}

	// One batch SELECT matching by name OR slug for maximum CSV compatibility.
	lookup, err := cs.categoryRepo.GetCategoriesByNameOrSlug(ctx, externalRefs)
	if err != nil {
		return nil, err
	}

	// Topological sort: each wave contains inputs whose parent is already
	// resolved. lookup grows as waves complete so child-of-child works.
	remaining := make([]*domain.BulkCategoryInput, len(inputs))
	copy(remaining, inputs)

	var ordered []*domain.Category

	for len(remaining) > 0 {
		var wave []*domain.BulkCategoryInput
		var still []*domain.BulkCategoryInput

		for _, in := range remaining {
			if in.ParentName == nil || *in.ParentName == "" {
				wave = append(wave, in)
			} else if _, ok := lookup[*in.ParentName]; ok {
				wave = append(wave, in)
			} else {
				still = append(still, in)
			}
		}

		if len(wave) == 0 {
			unresolved := make([]string, 0, len(still))
			for _, in := range still {
				unresolved = append(unresolved, *in.ParentName)
			}
			return nil, pkgerrors.NewValidation("UNRESOLVABLE_PARENTS",
				"parent categories not found: "+strings.Join(uniqueStrings(unresolved), ", "))
		}

		waveCategories := make([]*domain.Category, 0, len(wave))
		for _, in := range wave {
			cat := &domain.Category{
				ID:       uuid.New(),
				Name:     in.Name,
				Slug:     in.Slug,
				ParentID: in.ParentID,
			}
			if in.ParentName != nil && *in.ParentName != "" {
				if id, ok := lookup[*in.ParentName]; ok {
					cat.ParentID = &id
				}
			}
			waveCategories = append(waveCategories, cat)
		}

		persisted, err := cs.categoryRepo.BulkCreateCategories(ctx, waveCategories)
		if err != nil {
			return nil, err
		}

		// Index persisted rows by both name and slug for the next wave.
		for _, cat := range persisted {
			lookup[cat.Name] = cat.ID
			lookup[cat.Slug] = cat.ID
		}
		ordered = append(ordered, persisted...)
		remaining = still
	}

	cs.logger.Info("bulk created categories", "count", len(ordered))
	return ordered, nil
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

	cs.logger.Debug("fetching category by id", "id", id)

	return cs.categoryRepo.GetCategoryByID(ctx, id)
}

func (cs *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	if slug == "" {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "category slug is required")
	}

	cs.logger.Debug("fetching category by slug", "slug", slug)

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

	cs.logger.Debug("fetching category list", "limit", filters.Limit, "offset", filters.Offset)

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
