package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/interfaces/metrics"
)

// RateIngestor runs the scheduled rate ingestion on a ticker.
// It does NOT fetch on startup — it waits for the first tick.
type RateIngestor struct {
	ingest   *usecases.IngestRatesUseCase
	interval time.Duration
	base     string
	metrics  *metrics.Metrics
	log      *slog.Logger
}

func NewRateIngestor(
	ingest *usecases.IngestRatesUseCase,
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

// Run starts the background ticker. Blocks until ctx is cancelled.
func (r *RateIngestor) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	r.log.Info("rate ingestor started", "interval", r.interval, "base", r.base)
	for {
		select {
		case <-ctx.Done():
			r.log.Info("rate ingestor stopped")
			return
		case <-ticker.C:
			if err := r.ingest.Execute(ctx, r.base); err != nil {
				r.log.Warn("rate ingestion failed", "error", err)
				r.metrics.RecordIngest(false)
			} else {
				r.log.Info("rate ingestion succeeded", "base", r.base)
				r.metrics.RecordIngest(true)
			}
		}
	}
}
