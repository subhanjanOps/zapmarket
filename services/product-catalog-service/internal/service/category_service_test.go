package service_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// ── fake ─────────────────────────────────────────────────────────────────────

type fakeCategoryRepo struct {
	byID   map[uuid.UUID]*domain.Category
	bySlug map[string]*domain.Category
	byName map[string]*domain.Category
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{
		byID:   make(map[uuid.UUID]*domain.Category),
		bySlug: make(map[string]*domain.Category),
		byName: make(map[string]*domain.Category),
	}
}

func (r *fakeCategoryRepo) store(c *domain.Category) {
	r.byID[c.ID] = c
	r.bySlug[c.Slug] = c
	r.byName[c.Name] = c
}

func (r *fakeCategoryRepo) CreateCategory(_ context.Context, c *domain.Category) error {
	if _, ok := r.bySlug[c.Slug]; ok {
		return pkgerrors.NewConflict("SLUG_CONFLICT", "slug exists")
	}
	r.store(c)
	return nil
}
func (r *fakeCategoryRepo) BulkCreateCategories(_ context.Context, cats []*domain.Category) ([]*domain.Category, error) {
	for _, c := range cats {
		r.store(c)
	}
	return cats, nil
}
func (r *fakeCategoryRepo) GetCategoriesByNameOrSlug(_ context.Context, values []string) (map[string]uuid.UUID, error) {
	out := make(map[string]uuid.UUID)
	for _, v := range values {
		if c, ok := r.byName[v]; ok {
			out[v] = c.ID
		}
		if c, ok := r.bySlug[v]; ok {
			out[v] = c.ID
		}
	}
	return out, nil
}
func (r *fakeCategoryRepo) GetCategoryByID(_ context.Context, id uuid.UUID) (*domain.Category, error) {
	c, ok := r.byID[id]
	if !ok {
		return nil, pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "not found")
	}
	return c, nil
}
func (r *fakeCategoryRepo) GetCategoryBySlug(_ context.Context, slug string) (*domain.Category, error) {
	c, ok := r.bySlug[slug]
	if !ok {
		return nil, pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "not found")
	}
	return c, nil
}
func (r *fakeCategoryRepo) GetCategoryList(_ context.Context, _ *domain.CategoryFilters) ([]*domain.Category, int64, error) {
	var out []*domain.Category
	for _, c := range r.byID {
		out = append(out, c)
	}
	return out, int64(len(out)), nil
}
func (r *fakeCategoryRepo) UpdateCategory(_ context.Context, c *domain.Category) error {
	if _, ok := r.byID[c.ID]; !ok {
		return pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "not found")
	}
	r.store(c)
	return nil
}
func (r *fakeCategoryRepo) DeleteCategory(_ context.Context, id uuid.UUID) error {
	c, ok := r.byID[id]
	if !ok {
		return pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "not found")
	}
	delete(r.byID, id)
	delete(r.bySlug, c.Slug)
	delete(r.byName, c.Name)
	return nil
}

func newCategorySvc(repo *fakeCategoryRepo) service.CategoryService {
	return service.NewCategoryService(repo, slog.Default())
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestCreateCategory_Success(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	cat := &domain.Category{Name: "Electronics", Slug: "electronics"}
	if err := svc.CreateCategory(context.Background(), cat); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
}

func TestCreateCategory_MissingName(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	err := svc.CreateCategory(context.Background(), &domain.Category{Slug: "s"})
	assertValidation(t, err, "name")
}

func TestCreateCategory_MissingSlug(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	err := svc.CreateCategory(context.Background(), &domain.Category{Name: "N"})
	assertValidation(t, err, "slug")
}

func TestGetCategoryByID_NotFound(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	_, err := svc.GetCategoryByID(context.Background(), uuid.New())
	assertNotFound(t, err)
}

func TestGetCategoryByID_NilID(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	_, err := svc.GetCategoryByID(context.Background(), uuid.Nil)
	assertValidation(t, err, "id")
}

func TestGetCategoryBySlug_NotFound(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	_, err := svc.GetCategoryBySlug(context.Background(), "no-such")
	assertNotFound(t, err)
}

func TestUpdateCategory_MissingID(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	err := svc.UpdateCategory(context.Background(), &domain.Category{Name: "N", Slug: "s"})
	assertValidation(t, err, "id")
}

func TestDeleteCategory_NilID(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	err := svc.DeleteCategory(context.Background(), uuid.Nil)
	assertValidation(t, err, "id")
}

func TestGetCategoryList_InvalidSortField(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	_, _, err := svc.GetCategoryList(context.Background(), &domain.CategoryFilters{SortBy: "bad_field"})
	assertValidation(t, err, "sort_by")
}

func TestBulkCreateCategories_Empty(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	_, err := svc.BulkCreateCategories(context.Background(), nil)
	assertValidation(t, err, "empty")
}

func TestBulkCreateCategories_TooMany(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	inputs := make([]*domain.BulkCategoryInput, 501)
	for i := range inputs {
		inputs[i] = &domain.BulkCategoryInput{Name: "n", Slug: "s"}
	}
	_, err := svc.BulkCreateCategories(context.Background(), inputs)
	assertValidation(t, err, "limit")
}

func TestBulkCreateCategories_UnresolvableParent(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	parent := "nonexistent-parent"
	inputs := []*domain.BulkCategoryInput{
		{Name: "Child", Slug: "child", ParentName: &parent},
	}
	_, err := svc.BulkCreateCategories(context.Background(), inputs)
	if err == nil {
		t.Fatal("expected error for unresolvable parent")
	}
}

func TestBulkCreateCategories_ParentChildInSameBatch(t *testing.T) {
	svc := newCategorySvc(newFakeCategoryRepo())
	parentName := "Electronics"
	inputs := []*domain.BulkCategoryInput{
		{Name: "Electronics", Slug: "electronics"},
		{Name: "Laptops", Slug: "laptops", ParentName: &parentName},
	}
	cats, err := svc.BulkCreateCategories(context.Background(), inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}
	var child *domain.Category
	for _, c := range cats {
		if c.Name == "Laptops" {
			child = c
		}
	}
	if child == nil || child.ParentID == nil {
		t.Error("expected Laptops to have ParentID set")
	}
}
