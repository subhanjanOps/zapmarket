package contracts

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
)

// OrderListParams defines filters for admin order listing.
type OrderListParams struct {
	Status string
	UserID *uuid.UUID
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

// OrderPageParams is the minimal pagination input for buyer/seller order lists.
// Keeping it separate from OrderListParams avoids leaking admin filter fields
// into user-facing service contracts.
type OrderPageParams struct {
	Limit  int
	Offset int
}

// OrderRepository defines all persistence operations for the order saga.
// All mutations that transition order status also write an outbox row in the
// same DB transaction so no status change is observable without its event.
type OrderRepository interface {
	// GetByIdempotencyKey returns the existing order, or pkgerrors.NotFound.
	GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Order, error)

	// CreateOrder inserts the order row (status PENDING) and all items in one
	// DB transaction.
	CreateOrder(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error

	// GetByID returns pkgerrors.NotFound if no such order exists.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)

	// GetOrderItems returns all items for an order.
	GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderItem, error)

	// GetByUserID returns a paginated page of non-deleted orders for a user,
	// newest first, plus the total count for pagination metadata.
	GetByUserID(ctx context.Context, userID uuid.UUID, p OrderPageParams) ([]*domain.Order, int64, error)

	// GetBySellerID returns a paginated page of non-deleted orders that contain
	// at least one item with the given seller_id, newest first, plus total count.
	GetBySellerID(ctx context.Context, sellerID uuid.UUID, p OrderPageParams) ([]*domain.Order, int64, error)

	// MarkReserved sets order status → RESERVED and persists reservation IDs
	// on each item, all in one DB transaction.
	MarkReserved(ctx context.Context, orderID uuid.UUID, items []*domain.OrderItem) error

	// MarkConfirmed sets order status → CONFIRMED, records payment_id, and
	// writes the outbox event — all in one DB transaction.
	MarkConfirmed(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID, outboxPayload []byte) error

	// MarkCancelled sets order status → CANCELLED and writes the outbox event
	// in one DB transaction.
	MarkCancelled(ctx context.Context, orderID uuid.UUID, outboxPayload []byte) error

	// ListAll returns all orders with optional filters — admin use only.
	ListAll(ctx context.Context, params OrderListParams) ([]*domain.Order, int64, error)
}
