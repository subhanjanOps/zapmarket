package usecases

import (
	"context"
	"fmt"

	"github.com/zapmarket/zapmarket/services/currency-service/domain/repositories"
)

// ToggleCurrencyUseCase enables or disables a currency in the catalogue.
// Authorization (admin role) is enforced at the HTTP handler layer.
type ToggleCurrencyUseCase struct {
	repo repositories.CurrencyRepository
}

func NewToggleCurrencyUseCase(repo repositories.CurrencyRepository) *ToggleCurrencyUseCase {
	return &ToggleCurrencyUseCase{repo: repo}
}

func (uc *ToggleCurrencyUseCase) Execute(ctx context.Context, code string, enabled bool) error {
	if _, err := uc.repo.Get(ctx, code); err != nil {
		// Propagate domain errors (ErrCurrencyNotFound) unwrapped so callers can inspect them.
		return err
	}
	if err := uc.repo.Toggle(ctx, code, enabled); err != nil {
		return fmt.Errorf("toggle currency: %w", err)
	}
	return nil
}
