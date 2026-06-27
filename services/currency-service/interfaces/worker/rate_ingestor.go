package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/infrastructure/metrics"
)

const (
	backoffInitial = 5 * time.Second
	backoffMax     = 5 * time.Minute
	backoffFactor  = 2
)

// RateIngestor runs scheduled rate ingestion on a ticker.
// It fetches immediately on startup to warm the cache, then on every tick.
type RateIngestor struct {
	ingest   usecases.RateIngester
	interval time.Duration
	base     string
	metrics  *metrics.Metrics
	log      *slog.Logger
}

func NewRateIngestor(
	ingest usecases.RateIngester,
	interval time.Duration,
	base string,
	m *metrics.Metrics,
	log *slog.Logger,
) *RateIngestor {
	return &RateIngestor{
		ingest:   ingest,
		interval: interval,
		base:     base,
		metrics:  m,
		log:      log,
	}
}

// Run starts the background ingestor. Blocks until ctx is cancelled.
// An immediate fetch is performed before the first tick to ensure rates are available
// from the moment the service starts accepting traffic.
func (r *RateIngestor) Run(ctx context.Context) {
	r.log.Info("rate ingestor started", "interval", r.interval, "base", r.base)
	r.runOnce(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.log.Info("rate ingestor stopped")
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *RateIngestor) runOnce(ctx context.Context) {
	backoff := backoffInitial
	for {
		err := r.ingest.Execute(ctx, r.base)
		if err == nil {
			r.log.Info("rate ingestion succeeded", "base", r.base)
			r.metrics.RecordIngest(true)
			return
		}
		r.log.Warn("rate ingestion failed, retrying with backoff", "error", err, "backoff", backoff)
		r.metrics.RecordIngest(false)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= backoffFactor
		if backoff > backoffMax {
			backoff = backoffMax
		}
	}
}
