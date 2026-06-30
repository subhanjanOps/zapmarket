package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
)

type shipmentDeliveredEvent struct {
	ShipmentID string `json:"shipment_id"`
	OrderID    string `json:"order_id"`
}

type paymentLookup interface {
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error)
}

type captureMarker interface {
	HandleCaptureWebhook(ctx context.Context, paymentID uuid.UUID, gatewayTxnID string) error
}

// ShipmentConsumer listens on shipment.delivered and captures COD payments.
type ShipmentConsumer struct {
	repo        paymentLookup
	svc         captureMarker
	capturedPub publisher
	log         *slog.Logger
}

func NewShipmentConsumer(repo paymentLookup, svc captureMarker, capturedPub publisher, log *slog.Logger) *ShipmentConsumer {
	return &ShipmentConsumer{repo: repo, svc: svc, capturedPub: capturedPub, log: log}
}

func (c *ShipmentConsumer) Handle(ctx context.Context, msg pkgkafka.Message) error {
	var evt shipmentDeliveredEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		c.log.Error("shipment consumer: unmarshal failed", "error", err)
		return nil
	}

	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		c.log.Error("shipment consumer: invalid order_id", "order_id", evt.OrderID)
		return nil
	}

	payment, err := c.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		c.log.Error("shipment consumer: payment not found", "order_id", evt.OrderID, "error", err)
		return nil
	}

	if payment.Gateway != "cod" || payment.Status != domain.PaymentPendingCOD {
		// Not a COD order or already resolved — no-op.
		return nil
	}

	// COD payment: mark as captured. Use shipment_id as the "gateway txn id".
	if err := c.svc.HandleCaptureWebhook(ctx, payment.ID, "cod:"+evt.ShipmentID); err != nil {
		c.log.Error("shipment consumer: failed to capture COD payment", "payment_id", payment.ID, "error", err)
		return err
	}

	out := PaymentCapturedEvent{
		OrderID:     evt.OrderID,
		PaymentID:   payment.ID.String(),
		AmountCents: payment.Amount,
		Currency:    payment.Currency,
		CapturedAt:  time.Now().UTC(),
	}
	b, _ := json.Marshal(out)
	if pubErr := c.capturedPub.Publish(ctx, pkgkafka.Message{Key: []byte(evt.OrderID), Value: b}); pubErr != nil {
		c.log.Error("shipment consumer: failed to publish payment.captured", "error", pubErr)
	}

	c.log.Info("COD payment captured on delivery", "payment_id", payment.ID, "order_id", evt.OrderID)
	return nil
}
