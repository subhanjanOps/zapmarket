package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
)

//go:generate mockgen -source=repository_interfaces.go -destination=../mocks/repository_interfaces.go -package=mocks

// CategoryRepository defines the interface for category repository
type CategoryRepository interface {
	CreateCategory(ctx context.Context, category *domain.Category) error
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error)
	GetCategoryList(ctx context.Context, filters map[string]string, limit, offset int) ([]*domain.Category, error)
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

// ProductRepository defines the interface for product repository
type ProductRepository interface {
	CreateProduct(ctx context.Context, product *domain.Product) error
	GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
}

// SKURepository defines the interface for SKU repository
type SKURepository interface {
	CreateSku(ctx context.Context, sku *domain.SKU) error
	GetSkuByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error)
	GetSkuList(ctx context.Context, filters *domain.SKUFilters) ([]*domain.SKU, error)
	UpdateSku(ctx context.Context, sku *domain.SKU) error
	DeleteSku(ctx context.Context, id uuid.UUID) error
}

// ProductImageRepository defines the interface for product image repository
type ProductImageRepository interface {
	CreateProductImage(ctx context.Context, prdImage *domain.ProductImage) error
	GetImageByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error)
	GetImageBySKUID(ctx context.Context, skuID uuid.UUID) ([]*domain.ProductImage, error)
	UpdateProductImagePosition(ctx context.Context, id uuid.UUID, position int) error
	DeleteProductImage(ctx context.Context, id uuid.UUID) error
}
