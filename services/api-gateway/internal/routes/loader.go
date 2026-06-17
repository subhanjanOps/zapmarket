package routes

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
)

// Route is one row from gateway_routes.
type Route struct {
	ID          string
	PathPrefix  string
	Upstream    string
	AuthMode    string // "none" | "required" | "method_split"
	StripPrefix bool
	Enabled     bool
}

// Loader keeps an in-memory snapshot of enabled gateway_routes and refreshes
// it via Postgres LISTEN/NOTIFY plus a fallback poll every 30 s.
type Loader struct {
	db     *sql.DB
	dsn    string
	logger *slog.Logger

	mu      sync.RWMutex
	routes  []Route
	version atomic.Int64
}

func NewLoader(db *sql.DB, dsn string, logger *slog.Logger) *Loader {
	return &Loader{db: db, dsn: dsn, logger: logger}
}

// Load performs the initial synchronous load.
func (l *Loader) Load(ctx context.Context) error {
	return l.refresh(ctx)
}

// Routes returns a snapshot of the currently enabled routes.
func (l *Loader) Routes() []Route {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Route, len(l.routes))
	copy(out, l.routes)
	return out
}

// Version is bumped on every successful refresh; useful to detect changes.
func (l *Loader) Version() int64 { return l.version.Load() }

// Watch starts the background LISTEN/NOTIFY watcher and periodic poll.
// Returns when ctx is cancelled.
func (l *Loader) Watch(ctx context.Context) {
	go l.pollLoop(ctx)
	go l.listenLoop(ctx)
}

func (l *Loader) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := l.refresh(ctx); err != nil {
				l.logger.Warn("route poll refresh failed", "error", err)
			}
		}
	}
}

func (l *Loader) listenLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		l.runListener(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (l *Loader) runListener(ctx context.Context) {
	listener := pq.NewListener(l.dsn, 2*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			l.logger.Warn("pg listener event", "error", err)
		}
	})
	defer listener.Close()

	if err := listener.Listen("gateway_route_changed"); err != nil {
		l.logger.Warn("LISTEN failed", "error", err)
		return
	}
	l.logger.Info("listening for gateway_route_changed notifications")

	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-listener.Notify:
			if !ok {
				return
			}
			if n == nil {
				continue // keepalive ping
			}
			l.logger.Info("route change notified", "op", n.Extra)
			if err := l.refresh(ctx); err != nil {
				l.logger.Warn("route refresh after notify failed", "error", err)
			}
		case <-time.After(90 * time.Second):
			if err := listener.Ping(); err != nil {
				l.logger.Warn("pg listener ping failed", "error", err)
				return
			}
		}
	}
}

func (l *Loader) refresh(ctx context.Context) error {
	const q = `
		SELECT id, path_prefix, upstream, auth_mode, strip_prefix, enabled
		FROM gateway_routes
		WHERE enabled = true
		ORDER BY length(path_prefix) DESC, path_prefix`

	rows, err := l.db.QueryContext(ctx, q)
	if err != nil {
		return fmt.Errorf("query gateway_routes: %w", err)
	}
	defer rows.Close()

	var rs []Route
	for rows.Next() {
		var r Route
		if err := rows.Scan(&r.ID, &r.PathPrefix, &r.Upstream, &r.AuthMode, &r.StripPrefix, &r.Enabled); err != nil {
			return fmt.Errorf("scan route: %w", err)
		}
		rs = append(rs, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	l.mu.Lock()
	l.routes = rs
	l.mu.Unlock()
	l.version.Add(1)
	l.logger.Info("routes refreshed", "count", len(rs))
	return nil
}

// BuildDSN constructs the Postgres connection string from DB_* env vars.
func BuildDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("DB_USER", "zapuser"),
		getenv("DB_PASSWORD", "zappass123"),
		getenv("DB_HOST", "localhost"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "apigateway"),
	)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
