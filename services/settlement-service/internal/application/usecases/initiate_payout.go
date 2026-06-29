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

func (uc *InitiatePayoutUseCase) Execute(ctx context.Context, sellerID string) error {
	bal, err := uc.ledger.GetBalance(ctx, sellerID)
	if err != nil {
		return err
	}
	if bal.PendingPaise < uc.minBalancePaise {
		return domainerrors.ErrInsufficientBalance
	}
	_, err = uc.gateway.Initiate(ctx, sellerID, bal.PendingPaise, bal.Currency)
	if err != nil {
		return err
	}
	return uc.ledger.DebitBalance(ctx, sellerID, bal.PendingPaise)
}
