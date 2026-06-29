package application

import (
	"context"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

type CartRepository interface {
	GetCart(ctx context.Context, userID string) (*domain.Cart, error)
	UpsertItem(ctx context.Context, userID string, item domain.CartItem) error
	RemoveItem(ctx context.Context, userID, skuID string) error
	ClearCart(ctx context.Context, userID string) error
}

type SKUFetcher interface {
	GetSKUPrice(ctx context.Context, skuID string) (priceCents int64, currency string, inStock bool, err error)
}
