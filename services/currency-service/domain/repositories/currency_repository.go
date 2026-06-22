package repositories

import (
	"context"

	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

// CurrencyRepository reads and writes the currency catalogue.
type CurrencyRepository interface {
	ListEnabled(ctx context.Context) ([]entities.Currency, error)
	Get(ctx context.Context, code string) (entities.Currency, error)
	Toggle(ctx context.Context, code string, enabled bool) error
}
