package usecases

import (
	"context"
	"errors"
)

var ErrWishlistFull = errors.New("wishlist is at maximum capacity")

type wishlistRepo interface {
	CountByUser(ctx context.Context, userID string) (int, error)
	Add(ctx context.Context, userID, productID, skuID string) error
}

type AddToWishlistUseCase struct {
	repo     wishlistRepo
	maxItems int
}

func NewAddToWishlistUseCase(repo wishlistRepo, maxItems int) *AddToWishlistUseCase {
	return &AddToWishlistUseCase{repo: repo, maxItems: maxItems}
}

func (uc *AddToWishlistUseCase) Execute(ctx context.Context, userID, productID, skuID string) error {
	count, err := uc.repo.CountByUser(ctx, userID)
	if err != nil {
		return err
	}
	if count >= uc.maxItems {
		return ErrWishlistFull
	}
	return uc.repo.Add(ctx, userID, productID, skuID)
}
