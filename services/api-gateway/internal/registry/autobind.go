package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/zapmarket/zapmarket/services/api-gateway/internal/audit"
)

// swaggerDoc holds the subset of a Swagger 2.0 / OpenAPI 3.0 document we care about.
type swaggerDoc struct {
	// Swagger 2.0
	BasePath string `json:"basePath"`
	Host     string `json:"host"`
	// OpenAPI 3.0 fallback
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths map[string]map[string]struct {
		Security []map[string][]string `json:"security"`
	} `json:"paths"`
}

// routeSpec is the derived route we want to register for a service.
type routeSpec struct {
	PathPrefix  string
	AuthMode    string
	StripPrefix bool
}

// AutoBinder watches the Redis service registry for live instances and
// auto-registers their routes by fetching and parsing each service's Swagger spec.
// Enabled by GATEWAY_AUTO_BIND=true (default in development).
type AutoBinder struct {
	rdb        *goredis.Client
	db         *sql.DB
	audit      *audit.Writer
	logger     *slog.Logger
	httpClient *http.Client
	// seen tracks which (service, addr) pairs we've already fetched.
	// Prevents re-fetching the same spec on every tick.
	seen map[string]string // service name → last addr we successfully parsed
}

func NewAutoBinder(rdb *goredis.Client, db *sql.DB, aw *audit.Writer, logger *slog.Logger) *AutoBinder {
	return &AutoBinder{
		rdb:        rdb,
		db:         db,
		audit:      aw,
		logger:     logger,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		seen:       make(map[string]string),
	}
}

// Run scans for new service instances every 15 s and auto-binds their routes.
// Returns when ctx is cancelled.
func (ab *AutoBinder) Run(ctx context.Context) {
	ab.logger.Info("swagger auto-bind watcher started")
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
	// Find all live registry keys: svc:registry:{service}:{instance}
	// Uses SCAN instead of KEYS to avoid blocking Redis on large key sets.
	var allKeys []string
	var cursor uint64
	for {
		keys, next, err := ab.rdb.Scan(ctx, cursor, registryKeyPrefix+"*", 100).Result()
		if err != nil {
			ab.logger.Warn("auto-bind: redis scan failed", "error", err)
			return
		}
		allKeys = append(allKeys, keys...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	keys := allKeys

	// Group instances by service name. Use first healthy instance per service.
	seen := make(map[string]string) // service → addr
	for _, key := range keys {
		// key format: svc:registry:{service-name}:{instance-id}
		parts := strings.SplitN(strings.TrimPrefix(key, registryKeyPrefix), ":", 2)
		if len(parts) != 2 {
			continue
		}
		serviceName := parts[0]
		if _, already := seen[serviceName]; already {
			continue
		}
		raw, err := ab.rdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var inst Instance
		if err := json.Unmarshal(raw, &inst); err != nil {
			continue
		}
		seen[serviceName] = inst.Addr
	}

	for serviceName, addr := range seen {
		// Skip if we already processed this exact (service, addr) pair.
		if ab.seen[serviceName] == addr {
			continue
		}
		ab.processService(ctx, serviceName, addr)
	}
}

func (ab *AutoBinder) processService(ctx context.Context, serviceName, addr string) {
	specURL := addr + "/v1/docs/swagger.json"
	doc, err := ab.fetchSwagger(ctx, specURL)
	if err != nil {
		ab.logger.Warn("auto-bind: swagger fetch failed",
			"service", serviceName, "url", specURL, "error", err)
		return
	}

	routes := deriveRoutes(doc)
	if len(routes) == 0 {
		ab.logger.Warn("auto-bind: no routes derived from swagger",
			"service", serviceName, "url", specURL)
		return
	}

	for _, route := range routes {
		ab.applyRoute(ctx, serviceName, route)
	}

	// Mark as seen so we don't re-fetch until the address changes.
	ab.seen[serviceName] = addr
}

func (ab *AutoBinder) fetchSwagger(ctx context.Context, url string) (*swaggerDoc, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := ab.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("swagger endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB cap
	if err != nil {
		return nil, err
	}

	var doc swaggerDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse swagger: %w", err)
	}
	return &doc, nil
}

// deriveRoutes extracts one RouteSpec per logical path group from the swagger doc.
// For Swagger 2.0: basePath is the route prefix (e.g. /v1/auth).
// Auth mode is determined by scanning security fields across all operations.
func deriveRoutes(doc *swaggerDoc) []routeSpec {
	prefix := swaggerBasePath(doc)
	if prefix == "" || prefix == "/" {
		return nil
	}

	authMode := deriveAuthMode(doc)
	return []routeSpec{{
		PathPrefix:  prefix,
		AuthMode:    authMode,
		StripPrefix: false,
	}}
}

// swaggerBasePath extracts the canonical base path from the doc.
// Swagger 2.0: basePath field.
// OpenAPI 3.0: first server URL path component.
func swaggerBasePath(doc *swaggerDoc) string {
	if doc.BasePath != "" && doc.BasePath != "/" {
		// Normalise: strip trailing slash.
		return strings.TrimRight(doc.BasePath, "/")
	}
	if len(doc.Servers) > 0 {
		u := doc.Servers[0].URL
		// Extract just the path part if it's a full URL.
		if idx := strings.Index(u, "://"); idx != -1 {
			rest := u[idx+3:]
			if slash := strings.Index(rest, "/"); slash != -1 {
				path := strings.TrimRight(rest[slash:], "/")
				if path != "" && path != "/" {
					return path
				}
			}
		} else if strings.HasPrefix(u, "/") {
			return strings.TrimRight(u, "/")
		}
	}
	return ""
}

// deriveAuthMode analyses operations in the swagger doc and returns the most
// appropriate gateway auth_mode:
//
//   - "none"         — no operation has a security requirement, OR the security
//     pattern is mixed and cannot be cleanly expressed as a blanket rule
//   - "required"     — every operation has a security requirement
//   - "method_split" — GET/HEAD operations are all public; all write methods require auth
//
// When the pattern is mixed (some ops public, some secured, no clean method
// split), the gateway defaults to "none" and lets each service enforce its own
// per-endpoint auth. A blanket "required" in that case would block public
// endpoints like login/register that intentionally have no auth requirement.
func deriveAuthMode(doc *swaggerDoc) string {
	totalOps := 0
	securedOps := 0
	securedReadOps := 0
	totalReadOps := 0

	readMethods := map[string]bool{"get": true, "head": true}

	for _, methods := range doc.Paths {
		for method, op := range methods {
			totalOps++
			isRead := readMethods[strings.ToLower(method)]
			if isRead {
				totalReadOps++
			}
			if len(op.Security) > 0 {
				securedOps++
				if isRead {
					securedReadOps++
				}
			}
		}
	}

	if totalOps == 0 || securedOps == 0 {
		return "none"
	}
	if securedOps == totalOps {
		return "required"
	}
	// Clean method split: all reads are public, all writes are secured.
	if totalReadOps > 0 && securedReadOps == 0 && securedOps > 0 {
		return "method_split"
	}
	// Mixed pattern with no clean rule — let each service enforce its own auth.
	return "none"
}

// authLevel returns a numeric security level so we can compare modes.
// Higher = more restrictive.
func authLevel(mode string) int {
	switch mode {
	case "required":
		return 2
	case "method_split":
		return 1
	default: // "none" or unknown
		return 0
	}
}

// isAuthDowngrade returns true if newMode is less restrictive than current.
func isAuthDowngrade(current, newMode string) bool {
	return authLevel(newMode) < authLevel(current)
}

func (ab *AutoBinder) applyRoute(ctx context.Context, serviceName string, route routeSpec) {
	var existingUpstream, existingAuthMode string
	err := ab.db.QueryRowContext(ctx,
		`SELECT upstream, auth_mode FROM gateway_routes WHERE path_prefix = $1 AND enabled = true`,
		route.PathPrefix,
	).Scan(&existingUpstream, &existingAuthMode)

	switch {
	case err == sql.ErrNoRows:
		_, insertErr := ab.db.ExecContext(ctx,
			`INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (path_prefix) DO UPDATE
			   SET auth_mode = EXCLUDED.auth_mode
			 WHERE gateway_routes.upstream = EXCLUDED.upstream
			   AND gateway_routes.auth_mode != EXCLUDED.auth_mode`,
			route.PathPrefix, serviceName, route.AuthMode, route.StripPrefix,
		)
		if insertErr != nil {
			ab.logger.Warn("auto-bind: insert failed",
				"prefix", route.PathPrefix, "error", insertErr)
		} else {
			ab.logger.Info("auto-bind: route registered",
				"prefix", route.PathPrefix,
				"service", serviceName,
				"auth_mode", route.AuthMode)
		}
	case err != nil:
		ab.logger.Warn("auto-bind: db query failed", "error", err)
	case existingUpstream != serviceName:
		ab.logger.Error("auto-bind: route conflict",
			"prefix", route.PathPrefix,
			"existing", existingUpstream,
			"challenger", serviceName,
		)
		ab.audit.Log(audit.Entry{
			Event:    audit.EventRouteConflict,
			Path:     route.PathPrefix,
			Upstream: serviceName,
			Detail:   fmt.Sprintf("conflict: existing=%s challenger=%s", existingUpstream, serviceName),
		})
	default:
		// existingUpstream == serviceName. If auth_mode drifted, correct it —
		// but never downgrade from a more restrictive mode to a less restrictive
		// one (e.g. "required" → "none") since that would silently open a
		// previously-protected route. Operators must update auth_mode manually.
		if existingAuthMode != route.AuthMode && !isAuthDowngrade(existingAuthMode, route.AuthMode) {
			_, updateErr := ab.db.ExecContext(ctx,
				`UPDATE gateway_routes SET auth_mode = $1 WHERE path_prefix = $2 AND upstream = $3`,
				route.AuthMode, route.PathPrefix, serviceName,
			)
			if updateErr != nil {
				ab.logger.Warn("auto-bind: auth_mode update failed",
					"prefix", route.PathPrefix, "error", updateErr)
			} else {
				ab.logger.Info("auto-bind: auth_mode corrected",
					"prefix", route.PathPrefix,
					"old", existingAuthMode,
					"new", route.AuthMode)
			}
		} else if existingAuthMode != route.AuthMode {
			ab.logger.Warn("auto-bind: refusing auth_mode downgrade",
				"prefix", route.PathPrefix,
				"current", existingAuthMode,
				"proposed", route.AuthMode)
		}
	}
}
