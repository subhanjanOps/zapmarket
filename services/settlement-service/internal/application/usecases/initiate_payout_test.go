package usecases_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application/usecases"
	domainerrors "github.com/zapmarket/zapmarket/services/settlement-service/internal/domain/errors"
)

type fakePayoutGateway struct{ initiated string }

func (f *fakePayoutGateway) Initiate(_ context.Context, sellerID string, _ int64, _ string) (string, error) {
	f.initiated = sellerID
	return "rzp-pay-1", nil
}

func TestInitiatePayout_ExecutesWhenAboveMinimum(t *testing.T) {
	ledger := &fakeLedger{} // GetBalance returns 50000 paise = ₹500
	gw := &fakePayoutGateway{}
	uc := usecases.NewInitiatePayoutUseCase(ledger, gw, 10000) // min ₹100

	if err := uc.Execute(context.Background(), "sel-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.initiated != "sel-1" {
		t.Fatalf("expected payout for sel-1, got %q", gw.initiated)
	}
}

func TestInitiatePayout_ReturnsErrWhenBelowThreshold(t *testing.T) {
	ledger := &fakeLedger{} // returns 50000 paise
	gw := &fakePayoutGateway{}
	uc := usecases.NewInitiatePayoutUseCase(ledger, gw, 100000) // threshold above balance

	err := uc.Execute(context.Background(), "sel-1")
	if err != domainerrors.ErrInsufficientBalance {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}
