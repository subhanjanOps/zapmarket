package repositories

import (
	"context"
	"time"

	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain/entities"
)

type InventoryRepository interface {
	GetBySKU(ctx context.Context, skuID, warehouseID string) (*entities.Inventory, error)
	UpdateReserved(ctx context.Context, inventoryID string, delta int) error
}

type ReservationRepository interface {
	Create(ctx context.Context, r *entities.Reservation) error
	FindByID(ctx context.Context, id string) (*entities.Reservation, error)
	Release(ctx context.Context, id string) error
	FindExpired(ctx context.Context, before time.Time) ([]*entities.Reservation, error)
}

type LedgerRepository interface {
	Insert(ctx context.Context, entry *entities.LedgerEntry) error
}

type WarehouseRepository interface {
	FindByID(ctx context.Context, id string) (*entities.Warehouse, error)
	FindDefault(ctx context.Context) (*entities.Warehouse, error)
}
