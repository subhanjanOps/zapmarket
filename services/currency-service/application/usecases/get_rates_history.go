package usecases

import (
	"context"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/dto"
	domainerrors "github.com/zapmarket/zapmarket/services/currency-service/domain/errors"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

// HistoryRepository is a minimal read interface for historical rates (ISP).
type HistoryRepository interface {
	HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error)
}

type GetRatesHistoryUseCase struct {
	repo HistoryRepository
}

func NewGetRatesHistoryUseCase(repo HistoryRepository) *GetRatesHistoryUseCase {
	return &GetRatesHistoryUseCase{repo: repo}
}

func (uc *GetRatesHistoryUseCase) Execute(ctx context.Context, base string, date time.Time) (dto.RatesHistoryDTO, error) {
	rows, err := uc.repo.HistoryByBase(ctx, base, date)
	if err != nil {
		return dto.RatesHistoryDTO{}, err
	}
	if len(rows) == 0 {
		return dto.RatesHistoryDTO{}, &domainerrors.ErrNoRatesHistory{
			Base: base,
			Date: date.Format("2006-01-02"),
		}
	}
	rates := make(map[string]float64, len(rows))
	for _, r := range rows {
		rates[r.Quote] = r.Rate
	}
	return dto.RatesHistoryDTO{
		Base:  base,
		Date:  date.Format("2006-01-02"),
		Rates: rates,
	}, nil
}
