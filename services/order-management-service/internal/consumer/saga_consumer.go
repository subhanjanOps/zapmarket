// Package consumer contains Kafka consumer handlers for the checkout saga.
package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/events"
)

// orderStatusUpdater is the narrow repo interface the saga consumer needs.
type orderStatusUpdater interface {
	UpdateStatus(ctx context.Context, orderID uuid.UUID, status, sagaStatus string) error
}

// compensator releases held resources when a saga step fails.
type compensator interface {
	ReleaseStock(ctx context.Context, orderID string) error
	RefundPayment(ctx context.Context, orderID string) error
}

// eventPublisher publishes outcome events to Kafka topics.
type eventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
}

// SagaConsumer handles the terminal events of the checkout saga and drives
// the order to its final state (CONFIRMED or CANCELLED).
type SagaConsumer struct {
	repo   orderStatusUpdater
	comp   compensator
	pub    eventPublisher
	logger *slog.Logger
}

// NewSagaConsumer constructs a SagaConsumer.
func NewSagaConsumer(repo orderStatusUpdater, comp compensator, pub eventPublisher, logger *slog.Logger) *SagaConsumer {
	return &SagaConsumer{repo: repo, comp: comp, pub: pub, logger: logger}
}

// HandlePaymentCaptured transitions the order to CONFIRMED and publishes order.confirmed.
func (c *SagaConsumer) HandlePaymentCaptured(ctx context.Context, msg kafka.Message) error {
	var evt events.PaymentCapturedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return err
	}
	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		c.logger.ErrorContext(ctx, "saga: invalid order_id in payment.captured — skipping", "order_id", evt.OrderID, "error", err)
		return nil
	}
	if err := c.repo.UpdateStatus(ctx, orderID, "CONFIRMED", "COMPLETE"); err != nil {
		return err
	}
	confirmed := map[string]any{
		"order_id":     evt.OrderID,
		"payment_id":   evt.PaymentID,
		"confirmed_at": time.Now(),
	}
	b, err := json.Marshal(confirmed)
	if err != nil {
		return err
	}
	c.logger.InfoContext(ctx, "saga: order confirmed", "order_id", evt.OrderID, "payment_id", evt.PaymentID)
	return c.pub.Publish(ctx, kafka.TopicOrderConfirmed, evt.OrderID, b)
}

// HandleInventoryFailed transitions the order to CANCELLED and publishes order.cancelled.
// Inventory was never reserved so no compensation is needed.
func (c *SagaConsumer) HandleInventoryFailed(ctx context.Context, msg kafka.Message) error {
	var evt events.InventoryReservationFailedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return err
	}
	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		c.logger.ErrorContext(ctx, "saga: invalid order_id in inventory.reservation_failed — skipping", "order_id", evt.OrderID, "error", err)
		return nil
	}
	c.logger.WarnContext(ctx, "saga: inventory reservation failed — cancelling order", "order_id", evt.OrderID, "reason", evt.Reason)
	if err := c.repo.UpdateStatus(ctx, orderID, "CANCELLED", "COMPENSATED"); err != nil {
		return err
	}
	cancelled := map[string]any{
		"order_id":     evt.OrderID,
		"reason":       evt.Reason,
		"cancelled_at": time.Now(),
	}
	b, err := json.Marshal(cancelled)
	if err != nil {
		return err
	}
	return c.pub.Publish(ctx, kafka.TopicOrderCancelled, evt.OrderID, b)
}

// HandlePaymentFailed releases the inventory reservation, transitions the order
// to CANCELLED, and publishes order.cancelled.
func (c *SagaConsumer) HandlePaymentFailed(ctx context.Context, msg kafka.Message) error {
	var evt events.PaymentFailedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return err
	}
	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		c.logger.ErrorContext(ctx, "saga: invalid order_id in payment.failed — skipping", "order_id", evt.OrderID, "error", err)
		return nil
	}
	c.logger.WarnContext(ctx, "saga: payment failed — releasing inventory and cancelling order", "order_id", evt.OrderID, "reason", evt.Reason)

	// Release inventory even if it errors — log and continue to cancel.
	if err := c.comp.ReleaseStock(ctx, evt.OrderID); err != nil {
		c.logger.ErrorContext(ctx, "saga: release stock failed during compensation", "order_id", evt.OrderID, "error", err)
	}

	if err := c.repo.UpdateStatus(ctx, orderID, "CANCELLED", "COMPENSATED"); err != nil {
		return err
	}
	cancelled := map[string]any{
		"order_id":     evt.OrderID,
		"reason":       evt.Reason,
		"cancelled_at": time.Now(),
	}
	b, err := json.Marshal(cancelled)
	if err != nil {
		return err
	}
	return c.pub.Publish(ctx, kafka.TopicOrderCancelled, evt.OrderID, b)
}
