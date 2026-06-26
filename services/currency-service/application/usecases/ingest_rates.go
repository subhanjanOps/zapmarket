package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/repositories"
)

// IngestRatesUseCase fetches rates from the provider, persists them, warms the cache,
// and publishes a currency.rates.updated event.
type IngestRatesUseCase struct {
	provider  ports.RatesProvider
	ratesRepo repositories.RatesRepository
	cache     ports.RatesCache
	metrics   ports.MetricsRecorder
	cacheTTL  time.Duration
	publisher ports.EventPublisher
	log       *slog.Logger
}

func NewIngestRatesUseCase(
	provider ports.RatesProvider,
	ratesRepo repositories.RatesRepository,
	cache ports.RatesCache,
	cacheTTL time.Duration,
	m ports.MetricsRecorder,
	log *slog.Logger,
	publisher ports.EventPublisher,
) *IngestRatesUseCase {
	return &IngestRatesUseCase{
		provider:  provider,
		ratesRepo: ratesRepo,
		cache:     cache,
		metrics:   m,
		cacheTTL:  cacheTTL,
		publisher: publisher,
		log:       log,
	}
}

func (uc *IngestRatesUseCase) Execute(ctx context.Context, base string) error {
	fetchStart := time.Now()
	rateSet, err := uc.provider.FetchLatest(ctx, base)
	uc.metrics.RecordFetchDuration(fetchStart)
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

	payload, err := json.Marshal(map[string]any{
		"base":       base,
		"as_of":      rateSet.AsOf.Format("2006-01-02"),
		"rate_count": len(rows),
	})
	if err != nil {
		uc.log.Warn("ingest rates: marshal event payload failed", "error", err)
		return nil
	}

	if err := uc.publisher.Publish(ctx, "currency.rates.updated", payload); err != nil {
		uc.log.Warn("ingest rates: publish event failed", "error", err)
	}

	return nil
}
