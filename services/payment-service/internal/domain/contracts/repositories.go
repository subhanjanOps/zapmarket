// Package contracts defines the interfaces internal/service depends on.
// Concrete implementations live in internal/repository (DB) and
// internal/gateway (payment gateway); service code must depend only on
// these interfaces (see planning/01-foundation-hardening.md for why).
package contracts

//go:generate mockgen -source=repositories.go -destination=../../mocks/repository_mocks.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
)

// PaymentRepository defines the interface for payment persistence.
type PaymentRepository interface {
	// GetByIdempotencyKey returns the existing payment for key, or
	// pkgerrors.NotFound if none exists yet. ChargeCard calls this first so
	// a retried request returns the original result instead of charging
	// twice.
	GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Payment, error)

	// CreatePayment inserts a new payment row in `pending` status.
	CreatePayment(ctx context.Context, p *domain.Payment) error

	// MarkCaptured transitions a payment to `captured`, records the
	// gateway's transaction reference, and writes the two ledger.Entries
	// in the same DB transaction — a captured payment must never exist
	// without its ledger rows.
	MarkCaptured(ctx context.Context, paymentID uuid.UUID, gatewayTxnID string, entries []*domain.LedgerEntry) error

	// MarkFailed transitions a payment to `failed` and records why.
	MarkFailed(ctx context.Context, paymentID uuid.UUID, reason string) error

	// GetByID returns pkgerrors.NotFound if no such payment exists.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)

	// GetByOrderID returns the most recent payment for the given order.
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error)

	// CreateRefund inserts a refund row, updates the parent payment's
	// status (`refunded` or `partially_refunded`), writes the reversing
	// ledger entries, and publishes a `payment.refunded` outbox event —
	// all in one DB transaction. userID is required for the outbox payload.
	CreateRefund(ctx context.Context, refund *domain.Refund, userID uuid.UUID, newPaymentStatus domain.PaymentStatus, entries []*domain.LedgerEntry) error

	// GetByGatewayTxnID returns the payment matching a gateway transaction ID.
	GetByGatewayTxnID(ctx context.Context, txnID string) (*domain.Payment, error)
}

// ChargeResult is what a PaymentGateway returns for a successful charge.
type ChargeResult struct {
	GatewayTxnID string
}

// PaymentGateway defines the interface for the actual payment processor.
type PaymentGateway interface {
	// Name returns the gateway identifier stored on Payment records (e.g. "stripe", "fake").
	Name() string

	// Charge attempts to charge amount (in the smallest currency unit) and
	// returns a ChargeResult on success or an error on failure.
	// paymentMethodID is the gateway-specific token (e.g. Stripe pm_...); the
	// fake gateway ignores it.
	Charge(ctx context.Context, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*ChargeResult, error)

	// Refund attempts to refund amount against a previously-successful
	// charge identified by gatewayTxnID, returning the gateway's refund
	// reference on success.
	Refund(ctx context.Context, gatewayTxnID string, amount int64, currency string) (refundRef string, err error)
}
