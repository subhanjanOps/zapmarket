package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/application/ports"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain/entities"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain/repositories"
)

var ErrInsufficientStock = errors.New("insufficient stock to fulfill reservation")

type ReserveStockRequest struct {
	OrderID     string
	SKUID       string
	WarehouseID string
	Quantity    int
}

type ReserveStockResult struct {
	ReservationID string
}

type StockResult struct {
	SKUID        string
	WarehouseID  string
	QtyOnHand    int
	QtyReserved  int
	QtyAvailable int
}

type reserveStockUseCase struct {
	inventory    repositories.InventoryRepository
	reservations repositories.ReservationRepository
	ledger       repositories.LedgerRepository
	cache        ports.StockCache
}

func NewReserveStockUseCase(
	inv repositories.InventoryRepository,
	res repositories.ReservationRepository,
	led repositories.LedgerRepository,
	cache ports.StockCache,
) ReserveStockUseCase {
	return &reserveStockUseCase{inventory: inv, reservations: res, ledger: led, cache: cache}
}

func (uc *reserveStockUseCase) Execute(ctx context.Context, req ReserveStockRequest) (*ReserveStockResult, error) {
	ok, err := uc.cache.AtomicDecrement(ctx, req.SKUID, req.WarehouseID, req.Quantity)
	if err != nil || !ok {
		return nil, ErrInsufficientStock
	}

	reservation := &entities.Reservation{
		ID:        uuid.NewString(),
		OrderID:   req.OrderID,
		SKUID:     req.SKUID,
		Qty:       req.Quantity,
		Status:    "RESERVED",
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}
	if err := uc.reservations.Create(ctx, reservation); err != nil {
		return nil, err
	}
	return &ReserveStockResult{ReservationID: reservation.ID}, nil
}
