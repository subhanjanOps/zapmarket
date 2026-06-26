package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/infrastructure/cache"
)

// InventoryService defines the interface for inventory operations.
type InventoryService interface {
	AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error)
	ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error)
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error
	DeductStock(ctx context.Context, reservationID uuid.UUID) error
	GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error)
}

const stockCacheTTL = 24 * time.Hour

type inventoryService struct {
	repo   contracts.InventoryRepository
	cache  contracts.StockCachePort
	logger *slog.Logger
}

func NewInventoryService(repo contracts.InventoryRepository, stockCache contracts.StockCachePort, logger *slog.Logger) InventoryService {
	return &inventoryService{repo: repo, cache: stockCache, logger: logger}
}

func (s *inventoryService) AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error) {
	if skuID == uuid.Nil {
		return 0, pkgerrors.NewValidation("INVALID_DATA", "sku_id is required")
	}
	if qty <= 0 {
		return 0, pkgerrors.NewValidation("INVALID_DATA", "quantity must be greater than zero")
	}

	s.logger.Info("adding stock", "sku_id", skuID, "qty", qty)

	newQty, err := s.repo.AddStock(ctx, skuID, qty)
	if err != nil {
		return 0, err
	}

	// Increment Redis only if the key already exists. If absent, the next
	// ReserveStock cache-miss will warm it from DB with the correct value.
	if _, redisErr := s.cache.RunIncrIfExistsScript(ctx, s.cache.StockKey(skuID), qty); redisErr != nil {
		s.logger.Warn("failed to increment redis stock counter", "sku_id", skuID, "error", redisErr)
	}

	return newQty, nil
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

	key := s.cache.StockKey(skuID)

	result, err := s.cache.RunReserveScript(ctx, key, qty)
	if err != nil {
		s.logger.Warn("redis lua script error, falling through to db-only path", "error", err)
		return s.dbReserve(ctx, skuID, orderID, qty)
	}

	if result == -1 {
		// Cache miss: load from DB, warm Redis, then re-run Lua.
		inv, dbErr := s.repo.GetStock(ctx, skuID)
		if dbErr != nil {
			return nil, dbErr
		}
		if setErr := s.cache.Set(ctx, key, int64(inv.QtyAvailable), stockCacheTTL); setErr != nil {
			s.logger.Warn("failed to warm redis stock key", "sku_id", skuID, "error", setErr)
			return s.dbReserve(ctx, skuID, orderID, qty)
		}
		s.logger.Info("warmed redis stock cache", "sku_id", skuID, "available", inv.QtyAvailable)

		result, err = s.cache.RunReserveScript(ctx, key, qty)
		if err != nil {
			return s.dbReserve(ctx, skuID, orderID, qty)
		}
	}

	if result == 0 {
		return nil, pkgerrors.NewConflict("INSUFFICIENT_STOCK", "not enough stock available")
	}

	// Redis gate passed — now write the durable Postgres record.
	reservation, dbErr := s.repo.ReserveStock(ctx, skuID, orderID, qty)
	if dbErr != nil {
		if _, incrErr := s.cache.IncrBy(ctx, key, int64(qty)); incrErr != nil && !errors.Is(incrErr, cache.ErrCacheMiss) { //nolint:gosec
			s.logger.Error("CRITICAL: redis rollback failed after db reserve failure",
				"sku_id", skuID, "qty", qty, "error", incrErr)
		}
		return nil, dbErr
	}
	if reservation == nil {
		if _, incrErr := s.cache.IncrBy(ctx, key, int64(qty)); incrErr != nil && !errors.Is(incrErr, cache.ErrCacheMiss) { //nolint:gosec
			s.logger.Error("CRITICAL: redis rollback failed after nil reservation",
				"sku_id", skuID, "qty", qty, "error", incrErr)
		}
		return nil, pkgerrors.NewConflict("INSUFFICIENT_STOCK", "not enough stock available")
	}

	s.logger.Info("stock reserved", "sku_id", skuID, "reservation_id", reservation.ID, "redis_remaining", result)
	return reservation, nil
}

// dbReserve is the fallback path when Redis is unavailable: full check-and-
// decrement in Postgres (original single-DB behaviour, correct but not atomic).
func (s *inventoryService) dbReserve(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error) {
	reservation, err := s.repo.ReserveStock(ctx, skuID, orderID, qty)
	if err != nil {
		return nil, err
	}
	if reservation == nil {
		return nil, pkgerrors.NewConflict("INSUFFICIENT_STOCK", "not enough stock available")
	}
	return reservation, nil
}

func (s *inventoryService) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	if reservationID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "reservation_id is required")
	}

	s.logger.Info("releasing stock", "reservation_id", reservationID)

	skuID, qty, err := s.repo.ReleaseStock(ctx, reservationID)
	if err != nil {
		return err
	}

	if _, redisErr := s.cache.RunIncrIfExistsScript(ctx, s.cache.StockKey(skuID), int(qty)); redisErr != nil { //nolint:gosec
		s.logger.Warn("failed to increment redis stock after release", "reservation_id", reservationID, "error", redisErr)
	}

	return nil
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
