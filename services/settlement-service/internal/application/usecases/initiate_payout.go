package usecases

import (
	"context"

	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application"
	domainerrors "github.com/zapmarket/zapmarket/services/settlement-service/internal/domain/errors"
)

type InitiatePayoutUseCase struct {
	ledger          application.LedgerRepository
	gateway         application.PayoutGateway
	minBalancePaise int64
}

func NewInitiatePayoutUseCase(ledger application.LedgerRepository, gw application.PayoutGateway, minBalancePaise int64) *InitiatePayoutUseCase {
	return &InitiatePayoutUseCase{ledger: ledger, gateway: gw, minBalancePaise: minBalancePaise}
}

// Execute initiates a payout for sellerID. It first persists a PENDING
// payout row (which fails fast if one is already in flight for this seller,
// via a unique DB constraint), then calls the gateway, then marks the payout
// COMPLETED and debits the balance — so a crash between the gateway call and
// the debit leaves an auditable PENDING/FAILED payout instead of silently
// re-paying the seller on retry.
func (uc *InitiatePayoutUseCase) Execute(ctx context.Context, sellerID string) error {
	bal, err := uc.ledger.GetBalance(ctx, sellerID)
	if err != nil {
		return err
	}
	if bal.PendingPaise < uc.minBalancePaise {
		return domainerrors.ErrInsufficientBalance
	}

	payoutID, err := uc.ledger.CreatePendingPayout(ctx, sellerID, bal.PendingPaise, bal.Currency)
	if err != nil {
		return err
	}

	gatewayPayoutID, err := uc.gateway.Initiate(ctx, sellerID, bal.PendingPaise, bal.Currency)
	if err != nil {
		_ = uc.ledger.FailPayout(ctx, payoutID)
		return err
	}

	return uc.ledger.CompletePayout(ctx, payoutID, gatewayPayoutID, sellerID, bal.PendingPaise)
}
