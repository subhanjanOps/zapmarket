package ports

import (
	"context"
	"time"
)

// RateSet is a snapshot of rates fetched from an external provider.
type RateSet struct {
	Base  string
	AsOf  time.Time
	Rates map[string]float64
}

// RatesProvider fetches live exchange rates from an external source.
type RatesProvider interface {
	FetchLatest(ctx context.Context, base string) (RateSet, error)
}
