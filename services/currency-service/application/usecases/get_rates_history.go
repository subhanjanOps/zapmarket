package usecases

import (
	"context"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type RatesHistoryDTO struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

type HistoryRepository interface {
	HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error)
}

type GetRatesHistoryUseCase struct {
	repo HistoryRepository
}

func NewGetRatesHistoryUseCase(repo HistoryRepository) *GetRatesHistoryUseCase {
	return &GetRatesHistoryUseCase{repo: repo}
}

func (uc *GetRatesHistoryUseCase) Execute(ctx context.Context, base string, date time.Time) (RatesHistoryDTO, error) {
	rows, err := uc.repo.HistoryByBase(ctx, base, date)
	if err != nil {
		return RatesHistoryDTO{}, err
	}
	rates := make(map[string]float64, len(rows))
	for _, r := range rows {
		rates[r.Quote] = r.Rate
	}
	return RatesHistoryDTO{
		Base:  base,
		Date:  date.Format("2006-01-02"),
		Rates: rates,
	}, nil
}
