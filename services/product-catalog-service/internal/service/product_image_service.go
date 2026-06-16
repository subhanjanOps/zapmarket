package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain/contracts"
)

//go:generate mockgen -source=product_image_service.go -destination=../mocks/product_image_service.go -package=mocks

// allowedImageContentTypes is the allow-list of content types accepted for
// product image uploads. Anything else is rejected before it ever reaches
// object storage.
var allowedImageContentTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// UploadProductImageInput carries an in-flight file upload through to the
// service layer. Reader/Size/ContentType describe the actual file bytes;
// the content type here must already be the sniffed value (from
// http.DetectContentType), not a client-supplied header taken on faith.
type UploadProductImageInput struct {
	ProductID   uuid.UUID
	SKUID       *uuid.UUID
	Reader      io.Reader
	Size        int64
	ContentType string
}

// ProductImageService defines the interface for product image operations
type ProductImageService interface {
	UploadProductImage(ctx context.Context, in UploadProductImageInput) (*domain.ProductImage, error)
	GetImagesByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error)
	GetImagesBySKUID(ctx context.Context, skuID uuid.UUID) ([]*domain.ProductImage, error)
	UpdateImagePosition(ctx context.Context, id uuid.UUID, position int) error
	DeleteProductImage(ctx context.Context, id uuid.UUID) error
}

type productImageService struct {
	imageRepo contracts.ProductImageRepository
	storage   contracts.ObjectStorage
	logger    *slog.Logger
}

// NewProductImageService creates a new product image service
func NewProductImageService(repo contracts.ProductImageRepository, objectStorage contracts.ObjectStorage, logger *slog.Logger) ProductImageService {
	return &productImageService{
		imageRepo: repo,
		storage:   objectStorage,
		logger:    logger,
	}
}

func (pis *productImageService) UploadProductImage(ctx context.Context, in UploadProductImageInput) (*domain.ProductImage, error) {
	if in.ProductID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "product id is required")
	}

	ext, ok := allowedImageContentTypes[in.ContentType]
	if !ok {
		return nil, pkgerrors.NewValidation("INVALID_CONTENT_TYPE", "file must be one of: png, jpeg, webp, gif")
	}

	id := uuid.New()
	key := fmt.Sprintf("products/%s/%s%s", in.ProductID, id, ext)

	if _, err := pis.storage.Upload(ctx, key, in.Reader, in.Size, in.ContentType); err != nil {
		pis.logger.Error("failed to upload product image to object storage", "key", key, "error", err)
		return nil, pkgerrors.NewInternal("UPLOAD_FAILED", "failed to upload image", err)
	}

	image := &domain.ProductImage{
		ID:        id,
		ProductID: in.ProductID,
		SKUId:     in.SKUID,
		ObjectKey: key,
		URL:       pis.storage.PublicURL(key),
	}

	pis.logger.Info("uploading product image", "product_id", image.ProductID, "key", key)

	if err := pis.imageRepo.CreateProductImage(ctx, image); err != nil {
		// The DB write failed after the object was already stored — clean
		// up the orphan rather than leaving a file with no DB record.
		if delErr := pis.storage.Delete(ctx, key); delErr != nil {
			pis.logger.Error("failed to clean up orphaned object after DB write failure", "key", key, "error", delErr)
		}
		return nil, err
	}

	return image, nil
}

func (pis *productImageService) GetImagesByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error) {
	if productID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "product id is required")
	}

	pis.logger.Info("fetching images by product id", "product_id", productID)

	return pis.imageRepo.GetImageByProductID(ctx, productID)
}

func (pis *productImageService) GetImagesBySKUID(ctx context.Context, skuID uuid.UUID) ([]*domain.ProductImage, error) {
	if skuID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "sku id is required")
	}

	pis.logger.Info("fetching images by sku id", "sku_id", skuID)

	return pis.imageRepo.GetImageBySKUID(ctx, skuID)
}

func (pis *productImageService) UpdateImagePosition(ctx context.Context, id uuid.UUID, position int) error {
	if id == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "image id is required")
	}

	if position < 0 {
		return pkgerrors.NewValidation("INVALID_DATA", "position must be non-negative")
	}

	pis.logger.Info("updating image position", "id", id, "position", position)

	return pis.imageRepo.UpdateProductImagePosition(ctx, id, position)
}

func (pis *productImageService) DeleteProductImage(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "image id is required")
	}

	image, err := pis.imageRepo.GetImageByID(ctx, id)
	if err != nil {
		return err
	}

	pis.logger.Info("deleting product image", "id", id)

	if err := pis.imageRepo.DeleteProductImage(ctx, id); err != nil {
		return err
	}

	// Best-effort: an orphaned MinIO object is a cheap cleanup problem,
	// failing the whole delete because storage hiccuped is a worse one.
	if image.ObjectKey != "" {
		if err := pis.storage.Delete(ctx, image.ObjectKey); err != nil {
			pis.logger.Error("failed to delete object from storage after DB delete", "key", image.ObjectKey, "error", err)
		}
	}

	return nil
}
