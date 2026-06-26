package audit

import (
	"context"
	"database/sql"
	"log/slog"
	"sync/atomic"
	"time"
)

const (
	EventAuthRejected  = "AUTH_REJECTED"
	EventRateLimited   = "RATE_LIMITED"
	EventUpstream5xx   = "UPSTREAM_5XX"
	EventCircuitOpen   = "CIRCUIT_OPEN"
	EventRouteConflict = "ROUTE_CONFLICT"
)

// Entry is one audit event.
type Entry struct {
	RequestID  string
	UserID     string
	IP         string
	Method     string
	Path       string
	Upstream   string
	StatusCode int
	Event      string
	Detail     string
}

// Writer buffers audit entries and batch-inserts them to Postgres.
// Writes are fire-and-forget — the gateway never blocks on audit IO.
type Writer struct {
	db       *sql.DB
	ch       chan Entry
	logger   *slog.Logger
	done     chan struct{}
	dropped  atomic.Int64
}

// DroppedTotal returns the cumulative count of events dropped due to buffer
// overflow. Expose this via a Prometheus counter in the metrics handler.
func (w *Writer) DroppedTotal() int64 { return w.dropped.Load() }

const bufSize = 4096

func NewWriter(db *sql.DB, logger *slog.Logger) *Writer {
	return &Writer{
		db:     db,
		ch:     make(chan Entry, bufSize),
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Done returns a channel closed once Run has finished draining. Callers should
// wait on this before closing the database connection.
func (w *Writer) Done() <-chan struct{} {
	return w.done
}

// Log enqueues an entry. Drops with a counter increment if the buffer is full.
func (w *Writer) Log(e Entry) {
	select {
	case w.ch <- e:
	default:
		w.dropped.Add(1)
		w.logger.Warn("audit buffer full, dropping event", "event", e.Event, "total_dropped", w.dropped.Load())
	}
}

// Run drains the buffer and batch-inserts. Returns when ctx is cancelled.
func (w *Writer) Run(ctx context.Context) {
	defer close(w.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var batch []Entry
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := w.insert(ctx, batch); err != nil {
			w.logger.Warn("audit batch insert failed", "error", err, "count", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case e := <-w.ch:
			batch = append(batch, e)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			// Drain remaining entries on a fresh context — the lifecycle ctx
			// is already cancelled, so reusing it would fail every insert.
			drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
			for {
				select {
				case e := <-w.ch:
					batch = append(batch, e)
				default:
					if len(batch) > 0 {
						if err := w.insert(drainCtx, batch); err != nil {
							w.logger.Warn("audit drain insert failed", "error", err, "count", len(batch))
						}
						batch = batch[:0]
					}
					drainCancel()
					return
				}
			}
		}
	}
}

func (w *Writer) insert(ctx context.Context, batch []Entry) error {
	const q = `INSERT INTO gateway_audit_log
		(request_id, user_id, ip, method, path, upstream, status_code, event, detail)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, e := range batch {
		userID := sql.NullString{String: e.UserID, Valid: e.UserID != ""}
		_, err := stmt.ExecContext(ctx, e.RequestID, userID, e.IP, e.Method, e.Path,
			e.Upstream, e.StatusCode, e.Event, e.Detail)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
