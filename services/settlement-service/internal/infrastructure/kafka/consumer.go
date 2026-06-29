package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application/usecases"
)

type paymentCapturedEvent struct {
	OrderID     string `json:"order_id"`
	PaymentID   string `json:"payment_id"`
	SellerID    string `json:"seller_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

type paymentRefundedEvent struct {
	OrderID     string `json:"order_id"`
	PaymentID   string `json:"payment_id"`
	SellerID    string `json:"seller_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

type PaymentConsumer struct {
	credit *usecases.CreditSaleUseCase
	debit  *usecases.DebitRefundUseCase
	log    *slog.Logger
}

func NewPaymentConsumer(credit *usecases.CreditSaleUseCase, debit *usecases.DebitRefundUseCase, log *slog.Logger) *PaymentConsumer {
	return &PaymentConsumer{credit: credit, debit: debit, log: log}
}

func (c *PaymentConsumer) HandleCaptured(ctx context.Context, msg pkgkafka.Message) error {
	var evt paymentCapturedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		c.log.Error("settlement: unmarshal payment.captured failed", "error", err)
		return nil
	}
	c.log.Info("settlement: processing payment.captured", "order_id", evt.OrderID)
	return c.credit.Execute(ctx, usecases.CreditSaleInput{
		SellerID:    evt.SellerID,
		OrderID:     evt.OrderID,
		PaymentID:   evt.PaymentID,
		AmountPaise: evt.AmountCents,
		Currency:    evt.Currency,
	})
}

func (c *PaymentConsumer) HandleRefunded(ctx context.Context, msg pkgkafka.Message) error {
	var evt paymentRefundedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		c.log.Error("settlement: unmarshal payment.refunded failed", "error", err)
		return nil
	}
	c.log.Info("settlement: processing payment.refunded", "order_id", evt.OrderID)
	return c.debit.Execute(ctx, usecases.DebitRefundInput{
		SellerID:    evt.SellerID,
		OrderID:     evt.OrderID,
		PaymentID:   evt.PaymentID,
		AmountPaise: evt.AmountCents,
		Currency:    evt.Currency,
	})
}
