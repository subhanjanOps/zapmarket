package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
)

//go:generate mockgen -source=sku_service.go -destination=../mocks/sku_service.go -package=mocks

// SKUService defines the interface for SKU operations
type SKUService interface {
	CreateSKU(ctx context.Context, sku *domain.SKU) error
	GetSKUByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error)
	GetSKUList(ctx context.Context, filters *domain.SKUFilters) ([]*domain.SKU, error)
	UpdateSKU(ctx context.Context, sku *domain.SKU) error
	DeleteSKU(ctx context.Context, id uuid.UUID) error
}

type skuService struct {
	skuRepo  SKURepository
	logger   *slog.Logger
}

// NewSKUService creates a new SKU service
func NewSKUService(repo SKURepository, logger *slog.Logger) SKUService {
	return &skuService{
		skuRepo:  repo,
		logger:   logger,
	}
}

func (ss *skuService) CreateSKU(ctx context.Context, sku *domain.SKU) error {
	if sku.SKUCode == "" {
		return appErr.ValidationError("sku code is required", nil)
	}

	if sku.ProductID == uuid.Nil {
		return appErr.ValidationError("product id is required", nil)
	}

	if sku.PriceAmount <= 0 {
		return appErr.ValidationError("price amount must be greater than zero", nil)
	}

	if sku.Currency == "" {
		sku.Currency = "INR"
	}

	ss.logger.Info("creating sku", "sku_code", sku.SKUCode, "product_id", sku.ProductID)

	return ss.skuRepo.CreateSku(ctx, sku)
}

func (ss *skuService) GetSKUByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error) {
	if id == uuid.Nil {
		return nil, appErr.ValidationError("sku id is required", nil)
	}

	ss.logger.Info("fetching sku by id", "id", id)

	return ss.skuRepo.GetSkuByID(ctx, id)
}

func (ss *skuService) GetSKUList(ctx context.Context, filters *domain.SKUFilters) ([]*domain.SKU, error) {
	ss.logger.Info("fetching sku list", "filters", filters)

	return ss.skuRepo.GetSkuList(ctx, filters)
}

func (ss *skuService) UpdateSKU(ctx context.Context, sku *domain.SKU) error {
	if sku.ID == uuid.Nil {
		return appErr.ValidationError("sku id is required", nil)
	}

	if sku.SKUCode == "" {
		return appErr.ValidationError("sku code is required", nil)
	}

	if sku.ProductID == uuid.Nil {
		return appErr.ValidationError("product id is required", nil)
	}

	if sku.PriceAmount <= 0 {
		return appErr.ValidationError("price amount must be greater than zero", nil)
	}

	ss.logger.Info("updating sku", "id", sku.ID, "sku_code", sku.SKUCode)

	return ss.skuRepo.UpdateSku(ctx, sku)
}

func (ss *skuService) DeleteSKU(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return appErr.ValidationError("sku id is required", nil)
	}

	ss.logger.Info("deleting sku", "id", id)

	return ss.skuRepo.DeleteSku(ctx, id)
}
