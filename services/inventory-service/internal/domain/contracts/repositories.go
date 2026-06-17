// Package contracts defines the repository interface internal/service
// depends on. The concrete implementation lives in internal/repository;
// service code must depend only on this interface, never the concrete
// struct (see planning/01-foundation-hardening.md for why).
package contracts

import (
	"context"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
)

// InventoryRepository defines the interface for all inventory persistence
// operations. Every method that mutates qty_on_hand/qty_reserved also
// writes a domain.LedgerEntry in the same DB transaction — the ledger is the
// audit trail, and it must never be possible for a stock movement to commit
// without one.
type InventoryRepository interface {
	// AddStock increases qty_on_hand for sku_id at the default warehouse,
	// creating the inventory row if it doesn't exist, and returns the new
	// qty_on_hand.
	AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error)

	// ReserveStock atomically checks qty_available >= qty and, if so,
	// increments qty_reserved and creates a `reserved` Reservation row.
	// Returns (nil, nil) if there isn't enough stock — that's an expected
	// outcome, not an error.
	ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error)

	// ReleaseStock transitions a `reserved` reservation to `released` and
	// decrements qty_reserved by the same amount. Returns
	// pkgerrors.NotFound if the reservation doesn't exist, and a
	// pkgerrors.Conflict-typed error if it's not in `reserved` state
	// (already released or confirmed — releasing twice must not double
	// free stock).
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error

	// DeductStock transitions a `reserved` reservation to `confirmed` and
	// permanently removes the quantity from qty_on_hand (qty_reserved also
	// drops by the same amount, so qty_available is unaffected — it was
	// already reduced at reservation time).
	DeductStock(ctx context.Context, reservationID uuid.UUID) error

	// GetStock returns the current Inventory row for skuID at the default
	// warehouse. Returns pkgerrors.NotFound if no stock has ever been
	// added for this SKU.
	GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error)

	// GetReservationDetails returns the sku_id and qty for a reservation.
	// Used by ReleaseStock to know how much to add back to the Redis counter.
	GetReservationDetails(ctx context.Context, reservationID uuid.UUID) (skuID uuid.UUID, qty int, err error)
}
