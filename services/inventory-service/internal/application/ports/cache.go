package ports

import "context"

// StockCache provides atomic Redis operations for inventory hot path.
type StockCache interface {
	// AtomicDecrement decrements available qty if sufficient stock exists.
	// Returns false (not an error) if stock is insufficient.
	AtomicDecrement(ctx context.Context, skuID, warehouseID string, qty int) (bool, error)
	AtomicIncrement(ctx context.Context, skuID, warehouseID string, qty int) error
}
