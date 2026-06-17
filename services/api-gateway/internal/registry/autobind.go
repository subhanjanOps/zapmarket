package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/zapmarket/zapmarket/services/api-gateway/internal/audit"
)

// AutoBinder watches for service manifests in Redis and reconciles them
// against gateway_routes in Postgres. Enabled by GATEWAY_AUTO_BIND=true.
type AutoBinder struct {
	rdb    *goredis.Client
	db     *sql.DB
	audit  *audit.Writer
	logger *slog.Logger
}

func NewAutoBinder(rdb *goredis.Client, db *sql.DB, aw *audit.Writer, logger *slog.Logger) *AutoBinder {
	return &AutoBinder{rdb: rdb, db: db, audit: aw, logger: logger}
}

// Run scans for service manifests every 15 s and auto-inserts missing routes.
// Returns when ctx is cancelled.
func (ab *AutoBinder) Run(ctx context.Context) {
	ab.logger.Info("auto-bind watcher started")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ab.reconcile(ctx)
		}
	}
}

func (ab *AutoBinder) reconcile(ctx context.Context) {
	keys, err := ab.rdb.Keys(ctx, manifestKeyPrefix+"*").Result()
	if err != nil {
		ab.logger.Warn("auto-bind: redis scan failed", "error", err)
		return
	}

	for _, key := range keys {
		raw, err := ab.rdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			ab.logger.Warn("auto-bind: bad manifest", "key", key, "error", err)
			continue
		}
		ab.applyManifest(ctx, m)
	}
}

func (ab *AutoBinder) applyManifest(ctx context.Context, m Manifest) {
	for _, mr := range m.Routes {
		// Check for conflicts: does another service own this prefix?
		var existingUpstream string
		err := ab.db.QueryRowContext(ctx,
			`SELECT upstream FROM gateway_routes WHERE path_prefix = $1 AND enabled = true`,
			mr.PathPrefix,
		).Scan(&existingUpstream)

		switch {
		case err == sql.ErrNoRows:
			// No existing route — auto-insert.
			_, insertErr := ab.db.ExecContext(ctx,
				`INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
				 VALUES ($1, $2, $3, $4)
				 ON CONFLICT (path_prefix) DO NOTHING`,
				mr.PathPrefix, m.Service, mr.AuthMode, mr.StripPrefix,
			)
			if insertErr != nil {
				ab.logger.Warn("auto-bind: insert failed", "prefix", mr.PathPrefix, "error", insertErr)
			} else {
				ab.logger.Info("auto-bind: route registered", "prefix", mr.PathPrefix, "service", m.Service)
			}
		case err != nil:
			ab.logger.Warn("auto-bind: db query failed", "error", err)
		case existingUpstream != m.Service:
			// Conflict — different service already owns this prefix.
			ab.logger.Error("auto-bind: route conflict",
				"prefix", mr.PathPrefix,
				"existing", existingUpstream,
				"challenger", m.Service,
			)
			ab.audit.Log(audit.Entry{
				Event:  audit.EventRouteConflict,
				Path:   mr.PathPrefix,
				Detail: "conflict: " + existingUpstream + " vs " + m.Service,
			})
		}
		// existingUpstream == m.Service → already bound, no action needed.
	}
}
