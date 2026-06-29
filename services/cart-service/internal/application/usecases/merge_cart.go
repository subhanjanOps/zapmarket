package usecases

import (
	"context"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/application"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

type guestCartReader interface {
	GetGuestCart(ctx context.Context, sessionID string) (*domain.Cart, error)
	ClearGuestCart(ctx context.Context, sessionID string) error
}

type MergeCartUseCase struct {
	guest guestCartReader
	auth  application.CartRepository
	sku   application.SKUFetcher
}

func NewMergeCartUseCase(guest guestCartReader, auth application.CartRepository, sku application.SKUFetcher) *MergeCartUseCase {
	return &MergeCartUseCase{guest: guest, auth: auth, sku: sku}
}

func (uc *MergeCartUseCase) Execute(ctx context.Context, sessionID, userID string) error {
	guestCart, err := uc.guest.GetGuestCart(ctx, sessionID)
	if err != nil || len(guestCart.Items) == 0 {
		return err
	}
	for _, item := range guestCart.Items {
		price, currency, inStock, err := uc.sku.GetSKUPrice(ctx, item.SKUID)
		if err != nil || !inStock {
			continue
		}
		item.PriceAtAdd = price
		item.Currency = currency
		if err := uc.auth.UpsertItem(ctx, userID, item); err != nil {
			return err
		}
	}
	return uc.guest.ClearGuestCart(ctx, sessionID)
}
