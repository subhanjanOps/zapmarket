package usecases

import (
	"context"
	"time"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/application"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
	domainerrors "github.com/zapmarket/zapmarket/services/cart-service/internal/domain/errors"
)

type AddItemUseCase struct {
	repo application.CartRepository
	sku  application.SKUFetcher
}

func NewAddItemUseCase(repo application.CartRepository, sku application.SKUFetcher) *AddItemUseCase {
	return &AddItemUseCase{repo: repo, sku: sku}
}

func (uc *AddItemUseCase) Execute(ctx context.Context, userID string, item domain.CartItem) error {
	price, currency, inStock, err := uc.sku.GetSKUPrice(ctx, item.SKUID)
	if err != nil {
		return err
	}
	if !inStock {
		return domainerrors.ErrOutOfStock
	}
	item.PriceAtAdd = price
	item.Currency = currency
	item.AddedAt = time.Now()
	return uc.repo.UpsertItem(ctx, userID, item)
}
