package ports

import (
	"context"
	"time"
)

// CachedRates is the cache payload for exchange rates.
type CachedRates struct {
	Base  string
	AsOf  time.Time
	Rates map[string]float64
}

// RatesCache is an abstract read/write cache for the rates snapshot.
type RatesCache interface {
	Get(ctx context.Context, base string) (CachedRates, bool, error)
	Set(ctx context.Context, base string, r CachedRates, ttl time.Duration) error
}
