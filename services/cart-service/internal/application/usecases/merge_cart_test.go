package usecases_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

type fakeGuestRepo struct {
	cart    *domain.Cart
	cleared bool
}

func (f *fakeGuestRepo) GetGuestCart(_ context.Context, _ string) (*domain.Cart, error) {
	return f.cart, nil
}
func (f *fakeGuestRepo) ClearGuestCart(_ context.Context, _ string) error {
	f.cleared = true
	return nil
}

func TestMergeCart_MovesGuestItemsToAuthCart(t *testing.T) {
	guest := &fakeGuestRepo{
		cart: &domain.Cart{
			Items: []domain.CartItem{
				{SKUID: "sku-1", Quantity: 2, PriceAtAdd: 500, Currency: "INR"},
			},
		},
	}
	auth := &fakeCartRepo{}
	sku := &fakeSKUFetcher{price: 500, currency: "INR", inStock: true}

	uc := usecases.NewMergeCartUseCase(guest, auth, sku)
	if err := uc.Execute(context.Background(), "session-abc", "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.upserted.SKUID != "sku-1" {
		t.Fatalf("expected sku-1 merged, got %q", auth.upserted.SKUID)
	}
	if !guest.cleared {
		t.Fatal("expected guest cart to be cleared after merge")
	}
}
