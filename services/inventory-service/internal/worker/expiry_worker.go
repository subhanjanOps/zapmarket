// Package worker contains background workers for the inventory service.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
)

// reservationReleaser is a minimal interface so the worker can be tested
// without pulling in the full repository implementation.
type reservationReleaser interface {
	FindExpiredReservations(ctx context.Context, before time.Time) ([]*domain.Reservation, error)
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) (skuID uuid.UUID, qty int64, err error)
}

// ExpiryWorker polls for expired reservations and releases them, returning
// the stock to the available pool. It runs until its context is cancelled.
type ExpiryWorker struct {
	repo     reservationReleaser
	interval time.Duration
	log      *slog.Logger
}

// NewExpiryWorker creates an ExpiryWorker. interval is how often to poll;
// use 60*time.Second in production. log may be nil (a no-op logger is used).
func NewExpiryWorker(repo reservationReleaser, interval time.Duration, log *slog.Logger) *ExpiryWorker {
	if log == nil {
		log = slog.Default()
	}
	return &ExpiryWorker{repo: repo, interval: interval, log: log}
}

// Start runs the expiry loop, blocking until ctx is cancelled.
func (w *ExpiryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.log.Info("expiry worker started", "interval", w.interval)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("expiry worker stopped")
			return
		case t := <-ticker.C:
			w.runOnce(ctx, t)
		}
	}
}

func (w *ExpiryWorker) runOnce(ctx context.Context, now time.Time) {
	expired, err := w.repo.FindExpiredReservations(ctx, now)
	if err != nil {
		w.log.Error("expiry worker: find expired reservations failed", "error", err)
		return
	}

	for _, res := range expired {
		_, _, err := w.repo.ReleaseStock(ctx, res.ID)
		if err != nil {
			w.log.Error("expiry worker: release stock failed",
				"reservation_id", res.ID,
				"order_id", res.OrderID,
				"error", err,
			)
			continue
		}
		w.log.Info("expiry worker: released expired reservation",
			"reservation_id", res.ID,
			"order_id", res.OrderID,
			"sku_id", res.SKUID,
			"qty", res.Qty,
		)
	}
}
