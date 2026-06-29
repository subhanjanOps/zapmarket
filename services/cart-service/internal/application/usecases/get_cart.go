package usecases

import (
	"context"

	"github.com/zapmarket/zapmarket/services/cart-service/internal/application"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

type GetCartUseCase struct{ repo application.CartRepository }

func NewGetCartUseCase(repo application.CartRepository) *GetCartUseCase {
	return &GetCartUseCase{repo: repo}
}

func (uc *GetCartUseCase) Execute(ctx context.Context, userID string) (*domain.Cart, error) {
	return uc.repo.GetCart(ctx, userID)
}
