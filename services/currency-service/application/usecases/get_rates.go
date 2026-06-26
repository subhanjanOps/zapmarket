package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/dto"
	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/repositories"
)

// GetRatesUseCase serves exchange rates: Redis → DB. Returns 503-class error if rates exceed max age.
type GetRatesUseCase struct {
	ratesRepo       repositories.RatesRepository
	cache           ports.RatesCache
	metrics         ports.MetricsRecorder
	refreshInterval time.Duration
	maxAge          time.Duration
}

func NewGetRatesUseCase(
	ratesRepo repositories.RatesRepository,
	cache ports.RatesCache,
	m ports.MetricsRecorder,
	refreshInterval time.Duration,
	maxAge time.Duration,
) *GetRatesUseCase {
	return &GetRatesUseCase{
		ratesRepo:       ratesRepo,
		cache:           cache,
		metrics:         m,
		refreshInterval: refreshInterval,
		maxAge:          maxAge,
	}
}

// ErrRatesTooStale is returned when rates exceed MAX_RATE_AGE.
type ErrRatesTooStale struct{ Age time.Duration }

func (e *ErrRatesTooStale) Error() string {
	return fmt.Sprintf("exchange rates are too stale (age: %s)", e.Age)
}

func (uc *GetRatesUseCase) Execute(ctx context.Context, base string) (dto.RatesDTO, error) {
	// Try cache first.
	if cached, ok, err := uc.cache.Get(ctx, base); err == nil && ok {
		uc.metrics.RecordCacheHit()
		return uc.buildDTO(cached.Base, cached.AsOf, cached.Rates), nil
	}
	uc.metrics.RecordCacheMiss()

	// Fall back to DB.
	rates, asOf, err := uc.ratesRepo.LatestByBase(ctx, base)
	if err != nil {
		return dto.RatesDTO{}, fmt.Errorf("get rates: %w", err)
	}

	if len(rates) == 0 {
		return dto.RatesDTO{}, fmt.Errorf("no rates available for base %s", base)
	}

	age := time.Since(asOf)
	if age > uc.maxAge {
		return dto.RatesDTO{}, &ErrRatesTooStale{Age: age}
	}

	rateMap := make(map[string]float64, len(rates)+1)
	rateMap[base] = 1.0
	for _, r := range rates {
		rateMap[r.Quote] = r.Rate
	}
	return uc.buildDTO(base, asOf, rateMap), nil
}

func (uc *GetRatesUseCase) buildDTO(base string, asOf time.Time, rateMap map[string]float64) dto.RatesDTO {
	stale := time.Since(asOf) > uc.refreshInterval*2
	return dto.RatesDTO{
		Base:  base,
		AsOf:  asOf,
		Date:  asOf.Format("2006-01-02"),
		Stale: stale,
		Rates: rateMap,
	}
}
