package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

// orderCanceller is the repo subset needed by ReservationConsumer.
type orderCanceller interface {
	CancelIfPending(ctx context.Context, orderID uuid.UUID) error
}

// ReservationConsumer listens for reservation.expired and cancels the order if still PENDING.
type ReservationConsumer struct {
	repo   orderCanceller
	logger *slog.Logger
}

func NewReservationConsumer(repo orderCanceller, logger *slog.Logger) *ReservationConsumer {
	return &ReservationConsumer{repo: repo, logger: logger}
}

func (c *ReservationConsumer) Handle(ctx context.Context, msg pkgkafka.Message) error {
	var payload struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		c.logger.Warn("reservation consumer: bad payload", "error", err)
		return nil
	}
	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		c.logger.Warn("reservation consumer: invalid order_id", "order_id", payload.OrderID)
		return nil
	}
	if err := c.repo.CancelIfPending(ctx, orderID); err != nil {
		c.logger.Error("reservation consumer: failed to cancel order", "order_id", orderID, "error", err)
		return err
	}
	c.logger.Info("order cancelled due to reservation expiry", "order_id", orderID)
	return nil
}
