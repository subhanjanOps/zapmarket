package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain/contracts"
)

// InventoryService defines the interface for inventory operations.
type InventoryService interface {
	AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error)
	ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error)
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error
	DeductStock(ctx context.Context, reservationID uuid.UUID) error
	GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error)
}

type inventoryService struct {
	repo   contracts.InventoryRepository
	logger *slog.Logger
}

func NewInventoryService(repo contracts.InventoryRepository, logger *slog.Logger) InventoryService {
	return &inventoryService{repo: repo, logger: logger}
}

func (s *inventoryService) AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error) {
	if skuID == uuid.Nil {
		return 0, pkgerrors.NewValidation("INVALID_DATA", "sku_id is required")
	}
	if qty <= 0 {
		return 0, pkgerrors.NewValidation("INVALID_DATA", "quantity must be greater than zero")
	}

	s.logger.Info("adding stock", "sku_id", skuID, "qty", qty)

	return s.repo.AddStock(ctx, skuID, qty)
}

func (s *inventoryService) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error) {
	if skuID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "sku_id is required")
	}
	if orderID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "order_id is required")
	}
	if qty <= 0 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "quantity must be greater than zero")
	}

	s.logger.Info("reserving stock", "sku_id", skuID, "order_id", orderID, "qty", qty)

	return s.repo.ReserveStock(ctx, skuID, orderID, qty)
}

func (s *inventoryService) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	if reservationID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "reservation_id is required")
	}

	s.logger.Info("releasing stock", "reservation_id", reservationID)

	return s.repo.ReleaseStock(ctx, reservationID)
}

func (s *inventoryService) DeductStock(ctx context.Context, reservationID uuid.UUID) error {
	if reservationID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "reservation_id is required")
	}

	s.logger.Info("deducting stock", "reservation_id", reservationID)

	return s.repo.DeductStock(ctx, reservationID)
}

func (s *inventoryService) GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error) {
	if skuID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "sku_id is required")
	}

	return s.repo.GetStock(ctx, skuID)
}
