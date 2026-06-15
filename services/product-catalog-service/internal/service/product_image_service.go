package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

//go:generate mockgen -source=product_image_service.go -destination=../mocks/product_image_service.go -package=mocks

// ProductImageService defines the interface for product image operations
type ProductImageService interface {
	CreateProductImage(ctx context.Context, image *domain.ProductImage) error
	GetImagesByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error)
	GetImagesBySKUID(ctx context.Context, skuID uuid.UUID) ([]*domain.ProductImage, error)
	UpdateImagePosition(ctx context.Context, id uuid.UUID, position int) error
	DeleteProductImage(ctx context.Context, id uuid.UUID) error
}

type productImageService struct {
	imageRepo ProductImageRepository
	logger    *slog.Logger
}

// NewProductImageService creates a new product image service
func NewProductImageService(repo ProductImageRepository, logger *slog.Logger) ProductImageService {
	return &productImageService{
		imageRepo: repo,
		logger:    logger,
	}
}

func (pis *productImageService) CreateProductImage(ctx context.Context, image *domain.ProductImage) error {
	if image.ProductID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","product id is required")
	}

	if image.URL == "" {
		return pkgerrors.NewValidation("INVALID_DATA","image url is required")
	}

	pis.logger.Info("creating product image", "product_id", image.ProductID, "url", image.URL)

	return pis.imageRepo.CreateProductImage(ctx, image)
}

func (pis *productImageService) GetImagesByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error) {
	if productID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA","product id is required")
	}

	pis.logger.Info("fetching images by product id", "product_id", productID)

	return pis.imageRepo.GetImageByProductID(ctx, productID)
}

func (pis *productImageService) GetImagesBySKUID(ctx context.Context, skuID uuid.UUID) ([]*domain.ProductImage, error) {
	if skuID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA","sku id is required")
	}

	pis.logger.Info("fetching images by sku id", "sku_id", skuID)

	return pis.imageRepo.GetImageBySKUID(ctx, skuID)
}

func (pis *productImageService) UpdateImagePosition(ctx context.Context, id uuid.UUID, position int) error {
	if id == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","image id is required")
	}

	if position < 0 {
		return pkgerrors.NewValidation("INVALID_DATA","position must be non-negative")
	}

	pis.logger.Info("updating image position", "id", id, "position", position)

	return pis.imageRepo.UpdateProductImagePosition(ctx, id, position)
}

func (pis *productImageService) DeleteProductImage(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA","image id is required")
	}

	pis.logger.Info("deleting product image", "id", id)

	return pis.imageRepo.DeleteProductImage(ctx, id)
}
