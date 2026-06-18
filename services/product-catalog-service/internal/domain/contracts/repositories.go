// Package contracts defines the repository interfaces that internal/service
// depends on. Concrete implementations live in internal/repository; service
// code must depend only on these interfaces, never on the concrete structs.
package contracts

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
)

//go:generate mockgen -source=repositories.go -destination=../../mocks/repository_interfaces.go -package=mocks

// CategoryRepository defines the interface for category repository
type CategoryRepository interface {
	CreateCategory(ctx context.Context, category *domain.Category) error
	// BulkCreateCategories inserts all categories in a single transaction and
	// returns the persisted rows. On slug conflict the existing row is returned,
	// preserving its real ID for parent resolution.
	BulkCreateCategories(ctx context.Context, categories []*domain.Category) ([]*domain.Category, error)
	// GetCategoriesByNames returns id+name for every category whose name is in
	// the given set. Used by the bulk-import service to resolve parent_name →
	// parent_id without fetching the entire table.
	GetCategoriesByNames(ctx context.Context, names []string) (map[string]uuid.UUID, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error)
	// GetCategoryList returns the matching page of categories plus the total
	// count of rows matching filters (ignoring limit/offset), for pagination.
	GetCategoryList(ctx context.Context, filters *domain.CategoryFilters) ([]*domain.Category, int64, error)
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

// ProductRepository defines the interface for product repository
type ProductRepository interface {
	CreateProduct(ctx context.Context, product *domain.Product) error
	GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	// GetProductList returns the matching page of products plus the total
	// count of rows matching filters (ignoring limit/offset), for pagination.
	GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, int64, error)
	UpdateProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
}

// SKURepository defines the interface for SKU repository
type SKURepository interface {
	CreateSku(ctx context.Context, sku *domain.SKU) error
	GetSkuByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error)
	// GetSkuList returns the matching page of SKUs plus the total count of
	// rows matching filters (ignoring limit/offset), for pagination.
	GetSkuList(ctx context.Context, filters *domain.SKUFilters) ([]*domain.SKU, int64, error)
	UpdateSku(ctx context.Context, sku *domain.SKU) error
	DeleteSku(ctx context.Context, id uuid.UUID) error
}

// ProductImageRepository defines the interface for product image repository
type ProductImageRepository interface {
	CreateProductImage(ctx context.Context, prdImage *domain.ProductImage) error
	GetImageByID(ctx context.Context, id uuid.UUID) (*domain.ProductImage, error)
	GetImageByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error)
	GetImageBySKUID(ctx context.Context, skuID uuid.UUID) ([]*domain.ProductImage, error)
	UpdateProductImagePosition(ctx context.Context, id uuid.UUID, position int) error
	DeleteProductImage(ctx context.Context, id uuid.UUID) error
}

// ObjectStorage defines the interface for the object store backing
// user-uploaded files (product images today). Concrete implementation is
// pkg/storage.Client; the service layer depends only on this interface so
// it never imports the concrete MinIO/S3 SDK type directly.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}
