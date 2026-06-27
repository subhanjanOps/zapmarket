// Package gateway holds concrete contracts.PaymentGateway implementations.
package gateway

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain/contracts"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/refund"
)

// StripePaymentGateway implements contracts.PaymentGateway using Stripe.
type StripePaymentGateway struct{}

func NewStripePaymentGateway(secretKey string) *StripePaymentGateway {
	stripe.Key = secretKey
	return &StripePaymentGateway{}
}

func (g *StripePaymentGateway) Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*contracts.ChargeResult, error) {
	if paymentMethodID == "" {
		return nil, pkgerrors.NewValidation("PAYMENT_METHOD_REQUIRED", "payment_method_id is required for Stripe payments")
	}

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(amount),
		Currency:      stripe.String(currency),
		PaymentMethod: stripe.String(paymentMethodID),
		Confirm:       stripe.Bool(true),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled:        stripe.Bool(true),
			AllowRedirects: stripe.String("never"),
		},
	}
	params.IdempotencyKey = stripe.String(idempotencyKey.String())

	pi, err := paymentintent.New(params)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok {
			return nil, pkgerrors.NewValidation("CARD_DECLINED", stripeErr.Msg)
		}
		return nil, pkgerrors.NewInternal("PAYMENT_ERROR", fmt.Sprintf("stripe charge failed: %v", err), err)
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		return nil, pkgerrors.NewValidation("PAYMENT_INCOMPLETE", fmt.Sprintf("payment intent status: %s", pi.Status))
	}

	return &contracts.ChargeResult{GatewayTxnID: pi.ID}, nil
}

func (g *StripePaymentGateway) Refund(_ context.Context, gatewayTxnID string, amount int64, _ string) (string, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(gatewayTxnID),
		Amount:        stripe.Int64(amount),
	}

	r, err := refund.New(params)
	if err != nil {
		return "", pkgerrors.NewInternal("REFUND_ERROR", fmt.Sprintf("stripe refund failed: %v", err), err)
	}
	return r.ID, nil
}
