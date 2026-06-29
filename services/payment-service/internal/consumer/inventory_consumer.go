package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
)

// cardCharger is a narrow interface for charging payment cards.
type cardCharger interface {
	ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*domain.Payment, error)
}

// publisher is satisfied by *pkgkafka.Producer and test fakes.
type publisher interface {
	Publish(ctx context.Context, msg pkgkafka.Message) error
}

// InventoryConsumer handles inventory.reserved Kafka messages, charges the
// card via cardCharger.ChargeCard, and publishes payment.captured or
// payment.failed.
type InventoryConsumer struct {
	svc         cardCharger
	capturedPub publisher
	failedPub   publisher
	log         *slog.Logger
}

// NewInventoryConsumer creates an InventoryConsumer.
//   - svc        must implement cardCharger (ChargeCard method)
//   - capturedPub must publish to TopicPaymentCaptured
//   - failedPub   must publish to TopicPaymentFailed
func NewInventoryConsumer(
	svc cardCharger,
	capturedPub publisher,
	failedPub publisher,
	log *slog.Logger,
) *InventoryConsumer {
	return &InventoryConsumer{
		svc:         svc,
		capturedPub: capturedPub,
		failedPub:   failedPub,
		log:         log,
	}
}

// Handle satisfies pkgkafka.HandlerFunc and is called for every message on
// inventory.reserved.
func (c *InventoryConsumer) Handle(ctx context.Context, msg pkgkafka.Message) error {
	var evt InventoryReservedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		c.log.Error("inventory saga consumer: unmarshal failed", "error", err)
		// Malformed message — don't retry.
		return nil
	}

	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		c.log.Error("inventory saga consumer: invalid order_id", "order_id", evt.OrderID)
		return nil
	}

	userID, err := uuid.Parse(evt.UserID)
	if err != nil {
		c.log.Error("inventory saga consumer: invalid user_id", "user_id", evt.UserID, "order_id", evt.OrderID)
		return c.publishFailed(ctx, evt.OrderID, "invalid user_id: "+evt.UserID)
	}

	// Derive a deterministic idempotency key from the order ID so that
	// redelivered messages don't double-charge.
	idempotencyKey := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("saga:"+evt.OrderID))

	payment, chargeErr := c.svc.ChargeCard(
		ctx,
		orderID,
		userID,
		evt.AmountCents,
		evt.Currency,
		idempotencyKey,
		evt.PaymentMethodID,
	)
	if chargeErr != nil {
		c.log.Error("inventory saga consumer: charge failed",
			"order_id", evt.OrderID, "error", chargeErr)
		return c.publishFailed(ctx, evt.OrderID, chargeErr.Error())
	}

	// ChargeCard may return PaymentFailed status without an error when the
	// gateway declines. Treat that as a saga failure too.
	if payment.Status == domain.PaymentFailed || payment.FailureReason != nil {
		reason := "gateway declined"
		if payment.FailureReason != nil {
			reason = *payment.FailureReason
		}
		c.log.Error("inventory saga consumer: payment declined",
			"order_id", evt.OrderID, "reason", reason)
		return c.publishFailed(ctx, evt.OrderID, reason)
	}

	return c.publishCaptured(ctx, evt, payment.ID.String())
}

func (c *InventoryConsumer) publishCaptured(ctx context.Context, evt InventoryReservedEvent, paymentID string) error {
	out := PaymentCapturedEvent{
		OrderID:     evt.OrderID,
		PaymentID:   paymentID,
		AmountCents: evt.AmountCents,
		Currency:    evt.Currency,
		CapturedAt:  time.Now().UTC(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal PaymentCapturedEvent: %w", err)
	}
	return c.capturedPub.Publish(ctx, pkgkafka.Message{
		Key:   []byte(evt.OrderID),
		Value: b,
	})
}

func (c *InventoryConsumer) publishFailed(ctx context.Context, orderID, reason string) error {
	out := PaymentFailedEvent{
		OrderID:  orderID,
		Reason:   reason,
		FailedAt: time.Now().UTC(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal PaymentFailedEvent: %w", err)
	}
	return c.failedPub.Publish(ctx, pkgkafka.Message{
		Key:   []byte(orderID),
		Value: b,
	})
}
