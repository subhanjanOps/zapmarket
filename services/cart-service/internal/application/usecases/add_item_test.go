package usecases_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
	domainerrors "github.com/zapmarket/zapmarket/services/cart-service/internal/domain/errors"
)

type fakeCartRepo struct {
	cart     *domain.Cart
	upserted domain.CartItem
}

func (f *fakeCartRepo) GetCart(_ context.Context, _ string) (*domain.Cart, error) {
	if f.cart == nil {
		return &domain.Cart{}, nil
	}
	return f.cart, nil
}

func (f *fakeCartRepo) UpsertItem(_ context.Context, _ string, item domain.CartItem) error {
	f.upserted = item
	return nil
}

func (f *fakeCartRepo) RemoveItem(_ context.Context, _, _ string) error { return nil }
func (f *fakeCartRepo) ClearCart(_ context.Context, _ string) error     { return nil }

type fakeSKUFetcher struct {
	price    int64
	currency string
	inStock  bool
}

func (f *fakeSKUFetcher) GetSKUPrice(_ context.Context, _ string) (int64, string, bool, error) {
	return f.price, f.currency, f.inStock, nil
}

func TestAddItem_SnapshotsPriceAtAddTime(t *testing.T) {
	repo := &fakeCartRepo{}
	sku := &fakeSKUFetcher{price: 4999, currency: "INR", inStock: true}
	uc := usecases.NewAddItemUseCase(repo, sku)

	err := uc.Execute(context.Background(), "user-1", domain.CartItem{
		SKUID:    "sku-1",
		Quantity: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.upserted.PriceAtAdd != 4999 {
		t.Fatalf("expected price snapshot 4999, got %d", repo.upserted.PriceAtAdd)
	}
}

func TestAddItem_RejectsOutOfStockSKU(t *testing.T) {
	repo := &fakeCartRepo{}
	sku := &fakeSKUFetcher{price: 100, currency: "INR", inStock: false}
	uc := usecases.NewAddItemUseCase(repo, sku)

	err := uc.Execute(context.Background(), "user-1", domain.CartItem{SKUID: "sku-2", Quantity: 1})
	if err != domainerrors.ErrOutOfStock {
		t.Fatalf("expected ErrOutOfStock, got %v", err)
	}
}
