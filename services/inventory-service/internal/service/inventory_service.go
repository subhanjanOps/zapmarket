package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
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

// luaReserve atomically checks available qty and decrements by the requested
// amount. Returns:
//
//	-1  → key not in Redis (cache miss; caller must warm and retry)
//	 0  → insufficient stock
//	>0  → success; value is remaining available qty after decrement
var luaReserve = goredis.NewScript(`
local available = redis.call('GET', KEYS[1])
if available == false then return -1 end
available = tonumber(available)
local qty = tonumber(ARGV[1])
if available < qty then return 0 end
return redis.call('DECRBY', KEYS[1], qty)
`)

type inventoryService struct {
	repo   contracts.InventoryRepository
	rdb    *goredis.Client
	logger *slog.Logger
}

func NewInventoryService(repo contracts.InventoryRepository, rdb *goredis.Client, logger *slog.Logger) InventoryService {
	return &inventoryService{repo: repo, rdb: rdb, logger: logger}
}

func stockKey(skuID uuid.UUID) string {
	return fmt.Sprintf("inv:stock:%s", skuID)
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

	// Update Redis counter after the DB write succeeds.
	if redisErr := s.rdb.IncrBy(ctx, stockKey(skuID), int64(qty)).Err(); redisErr != nil {
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

	key := stockKey(skuID)

	// Lua check-and-decrement: atomic, no race between check and update.
	result, err := luaReserve.Run(ctx, s.rdb, []string{key}, strconv.Itoa(qty)).Int64()
	if err != nil && !errors.Is(err, goredis.Nil) {
		s.logger.Warn("redis lua script error, falling through to db-only path", "error", err)
		return s.dbReserve(ctx, skuID, orderID, qty)
	}

	if result == -1 {
		// Cache miss: load from DB, warm Redis, then re-run Lua.
		inv, dbErr := s.repo.GetStock(ctx, skuID)
		if dbErr != nil {
			return nil, dbErr
		}
		if setErr := s.rdb.Set(ctx, key, inv.QtyAvailable, 0).Err(); setErr != nil {
			s.logger.Warn("failed to warm redis stock key", "sku_id", skuID, "error", setErr)
			return s.dbReserve(ctx, skuID, orderID, qty)
		}
		s.logger.Info("warmed redis stock cache", "sku_id", skuID, "available", inv.QtyAvailable)

		result, err = luaReserve.Run(ctx, s.rdb, []string{key}, strconv.Itoa(qty)).Int64()
		if err != nil && !errors.Is(err, goredis.Nil) {
			return s.dbReserve(ctx, skuID, orderID, qty)
		}
	}

	if result == 0 {
		return nil, pkgerrors.NewConflict("INSUFFICIENT_STOCK", "not enough stock available")
	}

	// Redis gate passed — now write the durable Postgres record.
	reservation, dbErr := s.repo.ReserveStock(ctx, skuID, orderID, qty)
	if dbErr != nil {
		// Postgres rejected it (race at boundary or constraint violation) —
		// roll back the Redis decrement so the counters stay in sync.
		if incrErr := s.rdb.IncrBy(ctx, key, int64(qty)).Err(); incrErr != nil {
			s.logger.Error("CRITICAL: redis rollback failed after db reserve failure",
				"sku_id", skuID, "qty", qty, "error", incrErr)
		}
		return nil, dbErr
	}
	if reservation == nil {
		// DB check-and-update returned no rows (shouldn't happen after Lua pass,
		// but treat it as a race: roll back and report insufficient stock).
		if incrErr := s.rdb.IncrBy(ctx, key, int64(qty)).Err(); incrErr != nil {
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

	// Fetch reservation details before releasing so we can update Redis.
	skuID, qty, err := s.repo.GetReservationDetails(ctx, reservationID)
	if err != nil {
		return err
	}

	if err := s.repo.ReleaseStock(ctx, reservationID); err != nil {
		return err
	}

	// Increment Redis counter: released units are available again.
	if redisErr := s.rdb.IncrBy(ctx, stockKey(skuID), int64(qty)).Err(); redisErr != nil {
		s.logger.Warn("failed to increment redis stock after release", "reservation_id", reservationID, "error", redisErr)
	}

	return nil
}

func (s *inventoryService) DeductStock(ctx context.Context, reservationID uuid.UUID) error {
	if reservationID == uuid.Nil {
		return pkgerrors.NewValidation("INVALID_DATA", "reservation_id is required")
	}

	s.logger.Info("deducting stock", "reservation_id", reservationID)

	// DeductStock moves a reservation to CONFIRMED and reduces qty_on_hand.
	// Redis was already decremented when the reservation was created, so no
	// Redis update is needed here — available qty doesn't change again.
	return s.repo.DeductStock(ctx, reservationID)
}

func (s *inventoryService) GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error) {
	if skuID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "sku_id is required")
	}

	return s.repo.GetStock(ctx, skuID)
}
