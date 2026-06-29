package usecases

import (
	"context"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/application"
)

type RemoveItemUseCase struct{ repo application.CartRepository }

func NewRemoveItemUseCase(repo application.CartRepository) *RemoveItemUseCase {
	return &RemoveItemUseCase{repo: repo}
}

func (uc *RemoveItemUseCase) Execute(ctx context.Context, userID, skuID string) error {
	return uc.repo.RemoveItem(ctx, userID, skuID)
}
