// Package gateway holds concrete contracts.PaymentGateway implementations.
// FakePaymentGateway is the only one built so far — see
// planning/04-payment-service.md for why a real Razorpay/Stripe
// integration was deliberately deferred until there's an actual gateway
// account to wire credentials and webhook signatures against.
package gateway

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain/contracts"
)

// FakePaymentGateway simulates a payment processor for local dev and
// integration tests, with no network calls. It succeeds for any amount
// except one ending in 13 in the smallest currency unit (e.g. ₹X.13 for
// INR) — a deliberate, easy-to-trigger "magic amount" so saga compensation
// paths (Stage 5's Order Management) can be tested without a flaky/random
// failure mode.
type FakePaymentGateway struct{}

func NewFakePaymentGateway() *FakePaymentGateway {
	return &FakePaymentGateway{}
}

func (g *FakePaymentGateway) Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID, _ string) (*contracts.ChargeResult, error) {
	if amount%100 == 13 {
		return nil, pkgerrors.NewValidation("CARD_DECLINED", "the card was declined by the issuing bank")
	}

	return &contracts.ChargeResult{
		GatewayTxnID: fmt.Sprintf("fake_txn_%s", idempotencyKey),
	}, nil
}

func (g *FakePaymentGateway) Refund(ctx context.Context, gatewayTxnID string, amount int64, currency string) (string, error) {
	return fmt.Sprintf("fake_refund_%s_%d", gatewayTxnID, amount), nil
}
