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

func (f *fakeLedger) ApplyLedgerEntry(_ context.Context, e domain.LedgerEntry, delta int64) error {
	f.entries = append(f.entries, e)
	f.credited = delta
	return nil
}
func (f *fakeLedger) GetBalance(_ context.Context, _ string) (*domain.SellerBalance, error) {
	return &domain.SellerBalance{PendingPaise: 50000, Currency: "INR"}, nil
}
func (f *fakeLedger) GetPendingSellers(_ context.Context, _ int64) ([]string, error) {
	return nil, nil
}
func (f *fakeLedger) CreatePendingPayout(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return "payout-1", nil
}
func (f *fakeLedger) CompletePayout(_ context.Context, _, _, _ string, _ int64) error { return nil }
func (f *fakeLedger) FailPayout(_ context.Context, _ string) error                   { return nil }

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
	// net = 10000 - commission(200) - tds(100) - gst_on_commission(36) = 9664
	if ledger.credited != 9664 {
		t.Fatalf("expected net credited 9664, got %d", ledger.credited)
	}
	if ledger.entries[0].TDSPaise != 100 {
		t.Fatalf("expected TDS 100, got %d", ledger.entries[0].TDSPaise)
	}
	if ledger.entries[0].GSTOnCommissionPaise != 36 {
		t.Fatalf("expected GST on commission 36, got %d", ledger.entries[0].GSTOnCommissionPaise)
	}
}
