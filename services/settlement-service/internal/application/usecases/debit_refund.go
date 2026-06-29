package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type DebitRefundInput struct {
	SellerID    string
	OrderID     string
	PaymentID   string
	AmountPaise int64
	Currency    string
}

type DebitRefundUseCase struct{ ledger application.LedgerRepository }

func NewDebitRefundUseCase(ledger application.LedgerRepository) *DebitRefundUseCase {
	return &DebitRefundUseCase{ledger: ledger}
}

func (uc *DebitRefundUseCase) Execute(ctx context.Context, in DebitRefundInput) error {
	entry := domain.LedgerEntry{
		ID:          uuid.NewString(),
		SellerID:    in.SellerID,
		OrderID:     in.OrderID,
		PaymentID:   in.PaymentID,
		EntryType:   domain.EntryTypeDebitRefund,
		AmountPaise: -in.AmountPaise,
		NetPaise:    -in.AmountPaise,
		Currency:    in.Currency,
		CreatedAt:   time.Now(),
	}
	if err := uc.ledger.InsertEntry(ctx, entry); err != nil {
		return err
	}
	return uc.ledger.DebitBalance(ctx, in.SellerID, in.AmountPaise)
}
