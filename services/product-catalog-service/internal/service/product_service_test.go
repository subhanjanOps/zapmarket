package service_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeProductRepo struct {
	products map[uuid.UUID]*domain.Product
	slugs    map[string]*domain.Product
	err      error // injected error for all writes
}

func newFakeProductRepo() *fakeProductRepo {
	return &fakeProductRepo{
		products: make(map[uuid.UUID]*domain.Product),
		slugs:    make(map[string]*domain.Product),
	}
}

func (r *fakeProductRepo) CreateProduct(_ context.Context, p *domain.Product) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.slugs[p.Slug]; ok {
		return pkgerrors.NewConflict("SLUG_CONFLICT", "slug already exists")
	}
	r.products[p.ID] = p
	r.slugs[p.Slug] = p
	return nil
}
func (r *fakeProductRepo) GetProductByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	p, ok := r.products[id]
	if !ok {
		return nil, pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "not found")
	}
	return p, nil
}
func (r *fakeProductRepo) GetProductBySlug(_ context.Context, slug string) (*domain.Product, error) {
	p, ok := r.slugs[slug]
	if !ok {
		return nil, pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "not found")
	}
	return p, nil
}
func (r *fakeProductRepo) GetProductList(_ context.Context, filters *domain.ProductFilters) ([]*domain.Product, int64, error) {
	var out []*domain.Product
	for _, p := range r.products {
		out = append(out, p)
	}
	return out, int64(len(out)), nil
}
func (r *fakeProductRepo) UpdateProduct(_ context.Context, p *domain.Product) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.products[p.ID]; !ok {
		return pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "not found")
	}
	r.products[p.ID] = p
	return nil
}
func (r *fakeProductRepo) DeleteProduct(_ context.Context, id uuid.UUID) error {
	if _, ok := r.products[id]; !ok {
		return pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "not found")
	}
	delete(r.products, id)
	return nil
}

func newProductSvc(repo *fakeProductRepo) service.ProductService {
	return service.NewProductService(repo, slog.Default())
}

func validProduct() *domain.Product {
	return &domain.Product{
		Name:       "Wireless Mouse",
		Slug:       "wireless-mouse",
		CategoryID: uuid.New(),
		SellerID:   uuid.New(),
	}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestCreateProduct_Success(t *testing.T) {
	repo := newFakeProductRepo()
	svc := newProductSvc(repo)
	p := validProduct()
	if err := svc.CreateProduct(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if p.Status != domain.ProductStatusDraft {
		t.Errorf("expected draft status, got %s", p.Status)
	}
}

func TestCreateProduct_MissingName(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	p := validProduct()
	p.Name = ""
	err := svc.CreateProduct(context.Background(), p)
	assertValidation(t, err, "name")
}

func TestCreateProduct_MissingSlug(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	p := validProduct()
	p.Slug = ""
	err := svc.CreateProduct(context.Background(), p)
	assertValidation(t, err, "slug")
}

func TestCreateProduct_MissingCategory(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	p := validProduct()
	p.CategoryID = uuid.Nil
	err := svc.CreateProduct(context.Background(), p)
	assertValidation(t, err, "category")
}

func TestCreateProduct_MissingSeller(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	p := validProduct()
	p.SellerID = uuid.Nil
	err := svc.CreateProduct(context.Background(), p)
	assertValidation(t, err, "seller")
}

func TestGetProductByID_NotFound(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	_, err := svc.GetProductByID(context.Background(), uuid.New())
	assertNotFound(t, err)
}

func TestGetProductByID_NilID(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	_, err := svc.GetProductByID(context.Background(), uuid.Nil)
	assertValidation(t, err, "id")
}

func TestGetProductBySlug_NotFound(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	_, err := svc.GetProductBySlug(context.Background(), "missing-slug")
	assertNotFound(t, err)
}

func TestGetProductBySlug_EmptySlug(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	_, err := svc.GetProductBySlug(context.Background(), "")
	assertValidation(t, err, "slug")
}

func TestUpdateProduct_Success(t *testing.T) {
	repo := newFakeProductRepo()
	svc := newProductSvc(repo)
	p := validProduct()
	_ = svc.CreateProduct(context.Background(), p)

	p.Name = "Updated Mouse"
	if err := svc.UpdateProduct(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateProduct_MissingID(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	p := validProduct()
	p.ID = uuid.Nil
	err := svc.UpdateProduct(context.Background(), p)
	assertValidation(t, err, "id")
}

func TestDeleteProduct_Success(t *testing.T) {
	repo := newFakeProductRepo()
	svc := newProductSvc(repo)
	p := validProduct()
	_ = svc.CreateProduct(context.Background(), p)
	if err := svc.DeleteProduct(context.Background(), p.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteProduct_NilID(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	err := svc.DeleteProduct(context.Background(), uuid.Nil)
	assertValidation(t, err, "id")
}

func TestGetProductList_InvalidSortField(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	_, _, err := svc.GetProductList(context.Background(), &domain.ProductFilters{SortBy: "injected; DROP TABLE"})
	assertValidation(t, err, "sort_by")
}

func TestGetProductList_NilFilters(t *testing.T) {
	svc := newProductSvc(newFakeProductRepo())
	products, total, err := svc.GetProductList(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != int64(len(products)) {
		t.Errorf("total mismatch: got %d products but total=%d", len(products), total)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func assertValidation(t *testing.T, err error, _ string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Validation {
		t.Errorf("expected Validation error, got %T: %v", err, err)
	}
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.NotFound {
		t.Errorf("expected NotFound error, got %T: %v", err, err)
	}
}
