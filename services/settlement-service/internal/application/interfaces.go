package application

import (
	"context"

	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type LedgerRepository interface {
	// ApplyLedgerEntry inserts the ledger entry and adjusts the seller balance
	// by balanceDeltaPaise atomically in one transaction. If an entry already
	// exists for the same (payment_id, entry_type) — e.g. a Kafka redelivery —
	// it is a no-op: no duplicate row and no balance change.
	ApplyLedgerEntry(ctx context.Context, entry domain.LedgerEntry, balanceDeltaPaise int64) error
	GetBalance(ctx context.Context, sellerID string) (*domain.SellerBalance, error)
	GetPendingSellers(ctx context.Context, minBalancePaise int64) ([]string, error)

	// CreatePendingPayout records an in-flight payout. It fails if the seller
	// already has a PENDING payout (enforced by a unique partial index), so
	// concurrent scheduler runs cannot double-initiate a payout.
	CreatePendingPayout(ctx context.Context, sellerID string, amountPaise int64, currency string) (payoutID string, err error)
	CompletePayout(ctx context.Context, payoutID, gatewayPayoutID string, sellerID string, amountPaise int64) error
	FailPayout(ctx context.Context, payoutID string) error
}

type PayoutGateway interface {
	Initiate(ctx context.Context, sellerID string, amountPaise int64, currency string) (razorpayPayoutID string, err error)
}
