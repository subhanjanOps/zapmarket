package repositories

import (
	"context"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

// RatesRepository reads and writes exchange rates.
type RatesRepository interface {
	LatestByBase(ctx context.Context, base string) ([]entities.ExchangeRate, time.Time, error)
	UpsertLatest(ctx context.Context, rates []entities.ExchangeRate, asOf time.Time) error
	HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error)
}
