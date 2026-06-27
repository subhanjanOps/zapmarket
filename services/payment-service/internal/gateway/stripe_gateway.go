// Package gateway holds concrete contracts.PaymentGateway implementations.
package gateway

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain/contracts"

	"github.com/stripe/stripe-go/v82"
)

// StripePaymentGateway implements contracts.PaymentGateway using Stripe.
// It holds a per-instance stripe.Client so multiple gateways (e.g. in tests)
// do not clobber each other via the global stripe.Key.
type StripePaymentGateway struct {
	client *stripe.Client
}

func NewStripePaymentGateway(secretKey string) *StripePaymentGateway {
	return &StripePaymentGateway{client: stripe.NewClient(secretKey)}
}

func (g *StripePaymentGateway) Name() string { return "stripe" }

func (g *StripePaymentGateway) Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*contracts.ChargeResult, error) {
	if paymentMethodID == "" {
		return nil, pkgerrors.NewValidation("PAYMENT_METHOD_REQUIRED", "payment_method_id is required for Stripe payments")
	}

	params := &stripe.PaymentIntentCreateParams{
		Amount:        stripe.Int64(amount),
		Currency:      stripe.String(currency),
		PaymentMethod: stripe.String(paymentMethodID),
		Confirm:       stripe.Bool(true),
		AutomaticPaymentMethods: &stripe.PaymentIntentCreateAutomaticPaymentMethodsParams{
			Enabled:        stripe.Bool(true),
			AllowRedirects: stripe.String("never"),
		},
	}
	params.IdempotencyKey = stripe.String(idempotencyKey.String())

	pi, err := g.client.V1PaymentIntents.Create(ctx, params)
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

func (g *StripePaymentGateway) Refund(ctx context.Context, gatewayTxnID string, amount int64, _ string) (string, error) {
	params := &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(gatewayTxnID),
		Amount:        stripe.Int64(amount),
	}

	r, err := g.client.V1Refunds.Create(ctx, params)
	if err != nil {
		return "", pkgerrors.NewInternal("REFUND_ERROR", fmt.Sprintf("stripe refund failed: %v", err), err)
	}
	return r.ID, nil
}
