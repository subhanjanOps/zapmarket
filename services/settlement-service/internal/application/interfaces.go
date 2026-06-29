package application

import (
	"context"

	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type LedgerRepository interface {
	InsertEntry(ctx context.Context, entry domain.LedgerEntry) error
	CreditBalance(ctx context.Context, sellerID string, netPaise int64) error
	DebitBalance(ctx context.Context, sellerID string, amountPaise int64) error
	GetBalance(ctx context.Context, sellerID string) (*domain.SellerBalance, error)
	GetPendingSellers(ctx context.Context, minBalancePaise int64) ([]string, error)
}

type PayoutGateway interface {
	Initiate(ctx context.Context, sellerID string, amountPaise int64, currency string) (razorpayPayoutID string, err error)
}
