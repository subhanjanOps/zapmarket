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

	// Collect the parent names that must be resolved from the DB — only those
	// not already provided as a UUID and not satisfiable within this same batch.
	inputNames := make(map[string]struct{}, len(inputs))
	for _, in := range inputs {
		inputNames[in.Name] = struct{}{}
	}
	var externalNames []string
	for _, in := range inputs {
		if in.ParentName != nil && *in.ParentName != "" {
			if _, inBatch := inputNames[*in.ParentName]; !inBatch {
				externalNames = append(externalNames, *in.ParentName)
			}
		}
	}

	// One batch SELECT for all external parent names.
	nameToID, err := cs.categoryRepo.GetCategoriesByNames(ctx, externalNames)
	if err != nil {
		return nil, err
	}

	// Topological sort: each wave contains inputs whose parent is already
	// resolved (either from the DB lookup or from a prior wave's output).
	// nameToID grows as waves complete, so child-of-child works automatically.
	resolved := make(map[string]uuid.UUID) // name → new ID, built wave by wave
	remaining := make([]*domain.BulkCategoryInput, len(inputs))
	copy(remaining, inputs)

	var ordered []*domain.Category

	for len(remaining) > 0 {
		var wave []*domain.BulkCategoryInput
		var still []*domain.BulkCategoryInput

		for _, in := range remaining {
			if in.ParentName == nil || *in.ParentName == "" {
				// Root category — no parent needed.
				wave = append(wave, in)
			} else {
				pName := *in.ParentName
				if _, ok := nameToID[pName]; ok {
					wave = append(wave, in)
				} else if _, ok := resolved[pName]; ok {
					wave = append(wave, in)
				} else {
					still = append(still, in)
				}
			}
		}

		if len(wave) == 0 {
			// Remaining rows reference parent names that don't exist.
			names := make([]string, 0, len(still))
			for _, in := range still {
				names = append(names, *in.ParentName)
			}
			return nil, pkgerrors.NewValidation("UNRESOLVABLE_PARENTS",
				"parent categories not found: "+strings.Join(uniqueStrings(names), ", "))
		}

		// Build Category objects for this wave with resolved parent IDs.
		waveCategories := make([]*domain.Category, 0, len(wave))
		for _, in := range wave {
			cat := &domain.Category{
				ID:       uuid.New(),
				Name:     in.Name,
				Slug:     in.Slug,
				ParentID: in.ParentID,
			}
			if in.ParentName != nil && *in.ParentName != "" {
				pName := *in.ParentName
				if id, ok := nameToID[pName]; ok {
					cat.ParentID = &id
				} else if id, ok := resolved[pName]; ok {
					cat.ParentID = &id
				}
			}
			waveCategories = append(waveCategories, cat)
		}

		persisted, err := cs.categoryRepo.BulkCreateCategories(ctx, waveCategories)
		if err != nil {
			return nil, err
		}

		// Extend resolved map so the next wave can reference names created here.
		for _, cat := range persisted {
			resolved[cat.Name] = cat.ID
			nameToID[cat.Name] = cat.ID
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
