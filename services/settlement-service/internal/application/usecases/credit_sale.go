package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type CreditSaleInput struct {
	SellerID    string
	OrderID     string
	PaymentID   string
	AmountPaise int64
	Currency    string
}

type CreditSaleUseCase struct {
	ledger        application.LedgerRepository
	commissionBPS int64
}

func NewCreditSaleUseCase(ledger application.LedgerRepository, commissionBPS int64) *CreditSaleUseCase {
	return &CreditSaleUseCase{ledger: ledger, commissionBPS: commissionBPS}
}

func (uc *CreditSaleUseCase) Execute(ctx context.Context, in CreditSaleInput) error {
	commission := (in.AmountPaise * uc.commissionBPS) / 10000
	// TDS: 1% of gross sale amount (Section 194-O)
	tds := in.AmountPaise / 100
	// GST on platform commission: 18%
	gstOnCommission := (commission * 18) / 100
	net := in.AmountPaise - commission - tds - gstOnCommission

	entry := domain.LedgerEntry{
		ID:                   uuid.NewString(),
		SellerID:             in.SellerID,
		OrderID:              in.OrderID,
		PaymentID:            in.PaymentID,
		EntryType:            domain.EntryTypeCredit,
		AmountPaise:          in.AmountPaise,
		CommissionPaise:      commission,
		TDSPaise:             tds,
		GSTOnCommissionPaise: gstOnCommission,
		NetPaise:             net,
		Currency:             in.Currency,
		CreatedAt:            time.Now(),
	}
	if err := uc.ledger.InsertEntry(ctx, entry); err != nil {
		return err
	}
	return uc.ledger.CreditBalance(ctx, in.SellerID, net)
}
