package usecases

import "context"

type wishlistReader interface {
	List(ctx context.Context, userID string) ([]WishlistItemDTO, error)
}

// WishlistItemDTO decouples the usecase layer from the postgres row type.
type WishlistItemDTO struct {
	ProductID string
	SKUID     string
}

type wishlistWriter interface {
	Remove(ctx context.Context, userID, productID string) error
	Clear(ctx context.Context, userID string) error
}

type ListWishlistUseCase struct{ repo wishlistReader }

func NewListWishlistUseCase(repo wishlistReader) *ListWishlistUseCase {
	return &ListWishlistUseCase{repo: repo}
}

func (uc *ListWishlistUseCase) Execute(ctx context.Context, userID string) ([]WishlistItemDTO, error) {
	return uc.repo.List(ctx, userID)
}

type RemoveFromWishlistUseCase struct{ repo wishlistWriter }

func NewRemoveFromWishlistUseCase(repo wishlistWriter) *RemoveFromWishlistUseCase {
	return &RemoveFromWishlistUseCase{repo: repo}
}

func (uc *RemoveFromWishlistUseCase) Execute(ctx context.Context, userID, productID string) error {
	return uc.repo.Remove(ctx, userID, productID)
}

type ClearWishlistUseCase struct{ repo wishlistWriter }

func NewClearWishlistUseCase(repo wishlistWriter) *ClearWishlistUseCase {
	return &ClearWishlistUseCase{repo: repo}
}

func (uc *ClearWishlistUseCase) Execute(ctx context.Context, userID string) error {
	return uc.repo.Clear(ctx, userID)
}
