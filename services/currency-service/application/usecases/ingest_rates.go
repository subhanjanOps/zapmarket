package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/repositories"
)

// IngestRatesUseCase fetches rates from the provider, persists them, and warms the cache.
type IngestRatesUseCase struct {
	provider  ports.RatesProvider
	ratesRepo repositories.RatesRepository
	cache     ports.RatesCache
	cacheTTL  time.Duration
	log       *slog.Logger
}

func NewIngestRatesUseCase(
	provider ports.RatesProvider,
	ratesRepo repositories.RatesRepository,
	cache ports.RatesCache,
	cacheTTL time.Duration,
	log *slog.Logger,
) *IngestRatesUseCase {
	return &IngestRatesUseCase{
		provider:  provider,
		ratesRepo: ratesRepo,
		cache:     cache,
		cacheTTL:  cacheTTL,
		log:       log,
	}
}

func (uc *IngestRatesUseCase) Execute(ctx context.Context, base string) error {
	rateSet, err := uc.provider.FetchLatest(ctx, base)
	if err != nil {
		return fmt.Errorf("ingest rates: fetch: %w", err)
	}

	rows := make([]entities.ExchangeRate, 0, len(rateSet.Rates))
	for quote, rate := range rateSet.Rates {
		if quote == base {
			continue
		}
		rows = append(rows, entities.ExchangeRate{
			Base:  base,
			Quote: quote,
			Rate:  rate,
			AsOf:  rateSet.AsOf,
		})
	}

	if err := uc.ratesRepo.UpsertLatest(ctx, rows, rateSet.AsOf); err != nil {
		return fmt.Errorf("ingest rates: upsert: %w", err)
	}

	rateMap := make(map[string]float64, len(rateSet.Rates))
	rateMap[base] = 1.0
	for k, v := range rateSet.Rates {
		rateMap[k] = v
	}
	if err := uc.cache.Set(ctx, base, ports.CachedRates{
		Base:  base,
		AsOf:  rateSet.AsOf,
		Rates: rateMap,
	}, uc.cacheTTL); err != nil {
		uc.log.Warn("ingest rates: cache set failed", "error", err)
	}

	uc.log.Debug("rates ingested", "base", base, "count", len(rows), "as_of", rateSet.AsOf)
	return nil
}
