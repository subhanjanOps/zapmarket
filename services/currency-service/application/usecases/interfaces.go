package usecases

import (
	"context"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/dto"
)

// CurrencyLister lists enabled currencies from the catalogue.
type CurrencyLister interface {
	Execute(ctx context.Context) ([]dto.CurrencyDTO, error)
}

// RatesGetter returns the latest exchange rates for a given base currency.
type RatesGetter interface {
	Execute(ctx context.Context, base string) (dto.RatesDTO, error)
}

// CurrencyToggler enables or disables a currency in the catalogue.
type CurrencyToggler interface {
	Execute(ctx context.Context, code string, enabled bool) error
}

// RatesHistoryGetter returns historical exchange rates for a base currency and date.
type RatesHistoryGetter interface {
	Execute(ctx context.Context, base string, date time.Time) (dto.RatesHistoryDTO, error)
}

// RateIngester fetches rates from an external provider and persists them.
type RateIngester interface {
	Execute(ctx context.Context, base string) error
}
