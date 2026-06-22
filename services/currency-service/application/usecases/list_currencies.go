package usecases

import (
	"context"

	"github.com/zapmarket/zapmarket/services/currency-service/application/dto"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/repositories"
)

// ListCurrenciesUseCase returns the enabled currency catalogue.
type ListCurrenciesUseCase struct {
	repo repositories.CurrencyRepository
}

func NewListCurrenciesUseCase(repo repositories.CurrencyRepository) *ListCurrenciesUseCase {
	return &ListCurrenciesUseCase{repo: repo}
}

func (uc *ListCurrenciesUseCase) Execute(ctx context.Context) ([]dto.CurrencyDTO, error) {
	currencies, err := uc.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CurrencyDTO, len(currencies))
	for i, c := range currencies {
		result[i] = dto.CurrencyDTO{
			Code:     c.Code,
			Name:     c.Name,
			Flag:     c.Flag,
			Decimals: c.Decimals,
		}
	}
	return result, nil
}
