package audit

import (
	"context"
	"database/sql"
	"log/slog"
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
	db     *sql.DB
	ch     chan Entry
	logger *slog.Logger
}

const bufSize = 512

func NewWriter(db *sql.DB, logger *slog.Logger) *Writer {
	return &Writer{
		db:     db,
		ch:     make(chan Entry, bufSize),
		logger: logger,
	}
}

// Log enqueues an entry. Drops silently if the buffer is full (backpressure).
func (w *Writer) Log(e Entry) {
	select {
	case w.ch <- e:
	default:
		w.logger.Warn("audit buffer full, dropping event", "event", e.Event)
	}
}

// Run drains the buffer and batch-inserts. Returns when ctx is cancelled.
func (w *Writer) Run(ctx context.Context) {
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
			// Drain remaining entries.
			for {
				select {
				case e := <-w.ch:
					batch = append(batch, e)
				default:
					flush()
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
