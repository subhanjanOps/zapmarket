package clients

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
)

// orderItemsReader is the narrow repo interface SagaCompensator needs.
type orderItemsReader interface {
	GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderItem, error)
}

// inventoryReleaser is the narrow inventory interface SagaCompensator needs.
type inventoryReleaser interface {
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error
}

// SagaCompensator implements the consumer.compensator interface by looking up
// the order's reservation IDs and releasing them via gRPC.
type SagaCompensator struct {
	repo      orderItemsReader
	inventory inventoryReleaser
	logger    *slog.Logger
}

// NewSagaCompensator constructs a SagaCompensator.
func NewSagaCompensator(repo orderItemsReader, inventory inventoryReleaser, logger *slog.Logger) *SagaCompensator {
	return &SagaCompensator{repo: repo, inventory: inventory, logger: logger}
}

// ReleaseStock fetches the order items and releases each reservation.
// Errors on individual items are logged but do not abort the loop — partial
// releases are better than no release.
func (c *SagaCompensator) ReleaseStock(ctx context.Context, orderID string) error {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return err
	}
	items, err := c.repo.GetOrderItems(ctx, id)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ReservationID == nil {
			continue
		}
		if err := c.inventory.ReleaseStock(ctx, *item.ReservationID); err != nil {
			c.logger.ErrorContext(ctx, "saga compensator: failed to release reservation",
				"order_id", orderID,
				"reservation_id", item.ReservationID,
				"error", err,
			)
		}
	}
	return nil
}

// RefundPayment is a stub — payment refund via gRPC is not yet implemented in
// payment-service.  The call is logged so operators can trigger manual refunds.
func (c *SagaCompensator) RefundPayment(ctx context.Context, orderID string) error {
	c.logger.WarnContext(ctx, "saga compensator: payment refund not yet implemented — manual intervention required",
		"order_id", orderID)
	return nil
}
