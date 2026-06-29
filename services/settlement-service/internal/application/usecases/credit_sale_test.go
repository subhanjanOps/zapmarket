package usecases_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type fakeLedger struct {
	entries  []domain.LedgerEntry
	credited int64
}

func (f *fakeLedger) InsertEntry(_ context.Context, e domain.LedgerEntry) error {
	f.entries = append(f.entries, e)
	return nil
}
func (f *fakeLedger) CreditBalance(_ context.Context, _ string, net int64) error {
	f.credited = net
	return nil
}
func (f *fakeLedger) DebitBalance(_ context.Context, _ string, _ int64) error { return nil }
func (f *fakeLedger) GetBalance(_ context.Context, _ string) (*domain.SellerBalance, error) {
	return &domain.SellerBalance{PendingPaise: 50000, Currency: "INR"}, nil
}
func (f *fakeLedger) GetPendingSellers(_ context.Context, _ int64) ([]string, error) {
	return nil, nil
}

func TestCreditSale_DeductsCommissionAndCreditsNet(t *testing.T) {
	ledger := &fakeLedger{}
	uc := usecases.NewCreditSaleUseCase(ledger, 200) // 200 BPS = 2%

	err := uc.Execute(context.Background(), usecases.CreditSaleInput{
		SellerID:    "sel-1",
		OrderID:     "ord-1",
		PaymentID:   "pay-1",
		AmountPaise: 10000,
		Currency:    "INR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ledger.entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(ledger.entries))
	}
	if ledger.entries[0].CommissionPaise != 200 {
		t.Fatalf("expected commission 200, got %d", ledger.entries[0].CommissionPaise)
	}
	if ledger.credited != 9800 {
		t.Fatalf("expected net credited 9800, got %d", ledger.credited)
	}
}
