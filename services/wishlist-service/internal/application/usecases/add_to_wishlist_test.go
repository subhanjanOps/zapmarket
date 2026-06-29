package usecases_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/wishlist-service/internal/application/usecases"
)

type fakeWishlistRepo struct {
	count int
	saved bool
}

func (f *fakeWishlistRepo) CountByUser(_ context.Context, _ string) (int, error) { return f.count, nil }
func (f *fakeWishlistRepo) Add(_ context.Context, _, _, _ string) error {
	f.saved = true
	return nil
}

func TestAddToWishlist_EnforcesMaxLimit(t *testing.T) {
	repo := &fakeWishlistRepo{count: 200}
	uc := usecases.NewAddToWishlistUseCase(repo, 200)

	err := uc.Execute(context.Background(), "u1", "prod-1", "sku-1")
	if err == nil {
		t.Fatal("expected error when wishlist is full")
	}
}

func TestAddToWishlist_SavesWhenUnderLimit(t *testing.T) {
	repo := &fakeWishlistRepo{count: 5}
	uc := usecases.NewAddToWishlistUseCase(repo, 200)

	if err := uc.Execute(context.Background(), "u1", "prod-1", "sku-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.saved {
		t.Fatal("expected item saved to wishlist")
	}
}
