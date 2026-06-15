package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

//go:generate mockgen -source=product_service.go -destination=../mocks/product_service.go -package=mocks

// ProductService defines the interface for product operations
type ProductService interface {
	CreateProduct(ctx context.Context, product *domain.Product) error
	GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
}

type productService struct {
	productRepo ProductRepository
	logger      *slog.Logger
}

// NewProductService creates a new product service
func NewProductService(repo ProductRepository, logger *slog.Logger) ProductService {
	return &productService{
		productRepo: repo,
		logger:      logger,
	}
}

func (ps *productService) CreateProduct(ctx context.Context, product *domain.Product) error {
	if product.Name == "" {
		return pkgerrors.NewValidation("INVALID_DATA","product name is required")
	}

	if product.Slug == "" {
		return pkgerrors.NewValidation("INVALID_DATA","product slug is required")
	}

	if product.CategoryID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","category id is required")
	}

	if product.SellerID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","seller id is required")
	}

	if product.Status == "" {
		product.Status = domain.ProductStatusDraft
	}

	ps.logger.Info("creating product", "name", product.Name, "slug", product.Slug, "seller_id", product.SellerID)

	return ps.productRepo.CreateProduct(ctx, product)
}

func (ps *productService) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	if id == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA","product id is required")
	}

	ps.logger.Info("fetching product by id", "id", id)

	return ps.productRepo.GetProductByID(ctx, id)
}

func (ps *productService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	if slug == "" {
		return nil, pkgerrors.NewValidation("INVALID_DATA","product slug is required")
	}

	ps.logger.Info("fetching product by slug", "slug", slug)

	return ps.productRepo.GetProductBySlug(ctx, slug)
}

func (ps *productService) GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, error) {
	ps.logger.Info("fetching product list", "filters", filters)

	return ps.productRepo.GetProductList(ctx, filters)
}

func (ps *productService) UpdateProduct(ctx context.Context, product *domain.Product) error {
	if product.ID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","product id is required")
	}

	if product.Name == "" {
		return pkgerrors.NewValidation("INVALID_DATA","product name is required")
	}

	if product.Slug == "" {
		return pkgerrors.NewValidation("INVALID_DATA","product slug is required")
	}

	if product.CategoryID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","category id is required")
	}

	ps.logger.Info("updating product", "id", product.ID, "name", product.Name)

	return ps.productRepo.UpdateProduct(ctx, product)
}

func (ps *productService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","product id is required")
	}

	ps.logger.Info("deleting product", "id", id)

	return ps.productRepo.DeleteProduct(ctx, id)
}
