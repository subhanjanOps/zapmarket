package contracts

//go:generate mockgen -source=repositories.go -destination=../../mocks/repository_mocks.go -package=mocks

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
type OrderRepository interface {
	// GetByIdempotencyKey returns the existing order, or pkgerrors.NotFound.
	GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Order, error)

	// CreateOrder inserts the order row (status PENDING), all items, and the
	// checkout.requested outbox event in one DB transaction.
	CreateOrder(ctx context.Context, order *domain.Order, items []*domain.OrderItem, outboxPayload []byte) error

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

	// MarkConfirmed sets order status → CONFIRMED and records payment_id.
	// The saga consumer publishes order.confirmed directly to Kafka.
	MarkConfirmed(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID) error

	// MarkCancelled sets order status → CANCELLED.
	// The saga consumer publishes order.cancelled directly to Kafka.
	MarkCancelled(ctx context.Context, orderID uuid.UUID) error

	// SetItemReservationID persists the reservation_id for a single order item
	// identified by (orderID, skuID). Called when inventory.reserved is consumed.
	SetItemReservationID(ctx context.Context, orderID, skuID, reservationID uuid.UUID) error

	// ListAll returns all orders with optional filters — admin use only.
	ListAll(ctx context.Context, params OrderListParams) ([]*domain.Order, int64, error)

	// UpdateStatus sets the order status and saga_status columns directly.
	// Used by the saga consumer to confirm or cancel an order based on
	// downstream Kafka events (payment.captured / payment.failed / inventory.reservation_failed).
	// The saga consumer publishes the resulting event directly to Kafka.
	UpdateStatus(ctx context.Context, orderID uuid.UUID, status, sagaStatus string) error
}
