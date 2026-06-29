package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
)

// inventoryService is the subset of service.InventoryService used by the consumer.
type inventoryService interface {
	ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error)
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error
}

// publisher is the interface satisfied by *pkgkafka.Producer (and test fakes).
type publisher interface {
	Publish(ctx context.Context, msg pkgkafka.Message) error
}

// CheckoutConsumer handles checkout.requested Kafka messages, reserves stock
// for every item atomically, and publishes either inventory.reserved or
// inventory.reservation_failed.
type CheckoutConsumer struct {
	svc            inventoryService
	reservedPub    publisher
	reserveFailPub publisher
	log            *slog.Logger
}

// NewCheckoutConsumer creates a CheckoutConsumer.
//   - reservedPub    must publish to TopicInventoryReserved
//   - reserveFailPub must publish to TopicInventoryReservationFailed
func NewCheckoutConsumer(
	svc inventoryService,
	reservedPub publisher,
	reserveFailPub publisher,
	log *slog.Logger,
) *CheckoutConsumer {
	return &CheckoutConsumer{
		svc:            svc,
		reservedPub:    reservedPub,
		reserveFailPub: reserveFailPub,
		log:            log,
	}
}

// Handle satisfies pkgkafka.HandlerFunc and is called for every message on
// checkout.requested.
func (c *CheckoutConsumer) Handle(ctx context.Context, msg pkgkafka.Message) error {
	var evt CheckoutRequestedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		c.log.Error("checkout consumer: unmarshal failed", "error", err)
		// Malformed message — don't retry, just skip.
		return nil
	}

	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		c.log.Error("checkout consumer: invalid order_id", "order_id", evt.OrderID)
		return nil
	}

	var reserved []ReservationRef

	for _, item := range evt.Items {
		skuID, err := uuid.Parse(item.SKUID)
		if err != nil {
			c.log.Error("checkout consumer: invalid sku_id", "sku_id", item.SKUID, "order_id", evt.OrderID)
			c.compensate(ctx, reserved)
			return c.publishFailure(ctx, evt.OrderID, "invalid sku_id: "+item.SKUID)
		}

		res, err := c.svc.ReserveStock(ctx, skuID, orderID, item.Quantity)
		if err != nil {
			c.log.Error("checkout consumer: reserve failed",
				"order_id", evt.OrderID, "sku_id", item.SKUID, "error", err)
			c.compensate(ctx, reserved)
			return c.publishFailure(ctx, evt.OrderID, err.Error())
		}

		reserved = append(reserved, ReservationRef{
			SKUID:         item.SKUID,
			ReservationID: res.ID.String(),
			Quantity:      item.Quantity,
		})
	}

	return c.publishReserved(ctx, evt.OrderID, reserved)
}

// compensate releases all already-reserved items when a later item fails.
func (c *CheckoutConsumer) compensate(ctx context.Context, reserved []ReservationRef) {
	for _, r := range reserved {
		resID, err := uuid.Parse(r.ReservationID)
		if err != nil {
			c.log.Error("checkout consumer: compensate — bad reservation_id", "reservation_id", r.ReservationID)
			continue
		}
		if err := c.svc.ReleaseStock(ctx, resID); err != nil {
			c.log.Error("checkout consumer: compensate — release failed",
				"reservation_id", r.ReservationID, "error", err)
		}
	}
}

func (c *CheckoutConsumer) publishReserved(ctx context.Context, orderID string, refs []ReservationRef) error {
	out := InventoryReservedEvent{
		OrderID:      orderID,
		Reservations: refs,
		ReservedAt:   time.Now().UTC(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return c.reservedPub.Publish(ctx, pkgkafka.Message{
		Key:   []byte(orderID),
		Value: b,
	})
}

func (c *CheckoutConsumer) publishFailure(ctx context.Context, orderID, reason string) error {
	out := InventoryReservationFailedEvent{
		OrderID:  orderID,
		Reason:   reason,
		FailedAt: time.Now().UTC(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return c.reserveFailPub.Publish(ctx, pkgkafka.Message{
		Key:   []byte(orderID),
		Value: b,
	})
}
