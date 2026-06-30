package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

// orderDeliveryUpdater is the repo subset needed by ShipmentConsumer.
type orderDeliveryUpdater interface {
	UpdateStatus(ctx context.Context, orderID uuid.UUID, status, sagaStatus string) error
}

// ShipmentConsumer listens for shipment.undelivered and marks the order DELIVERY_FAILED.
type ShipmentConsumer struct {
	repo   orderDeliveryUpdater
	logger *slog.Logger
}

func NewShipmentConsumer(repo orderDeliveryUpdater, logger *slog.Logger) *ShipmentConsumer {
	return &ShipmentConsumer{repo: repo, logger: logger}
}

func (c *ShipmentConsumer) Handle(ctx context.Context, msg pkgkafka.Message) error {
	var payload struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		c.logger.Warn("shipment consumer: bad payload", "error", err)
		return nil
	}
	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		c.logger.Warn("shipment consumer: invalid order_id", "order_id", payload.OrderID)
		return nil
	}
	if err := c.repo.UpdateStatus(ctx, orderID, "DELIVERY_FAILED", ""); err != nil {
		c.logger.Error("shipment consumer: failed to update order status", "order_id", orderID, "error", err)
		return err
	}
	c.logger.Info("order marked DELIVERY_FAILED", "order_id", orderID)
	return nil
}
