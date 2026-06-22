package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/admin"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/audit"
	gw "github.com/zapmarket/zapmarket/services/api-gateway/internal/middleware"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/metrics"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/proxy"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/registry"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/routes"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	log := logger.New(cfg.AppEnv)

	// ── Gateway DB (dedicated apigateway database) ─────────────────────────
	db, err := sql.Open("postgres", routes.BuildDSN())
	if err != nil {
		log.Error("failed to open gateway DB", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(context.Background()); err != nil {
		log.Error("gateway DB unreachable", "error", err)
		os.Exit(1)
	}
	log.Info("connected to gateway DB")

	if cfg.MigrateOnBoot {
		if err := migrate.Up(cfg, "migrations"); err != nil {
			log.Error("migration failed", "error", err)
			os.Exit(1)
		}
	}

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("Redis unreachable", "addr", cfg.RedisURL, "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to Redis", "addr", cfg.RedisURL)

	// ── Auth middleware ────────────────────────────────────────────────────
	authMW, err := gw.NewAuthMiddleware(cfg.AuthServiceAddr)
	if err != nil {
		log.Error("failed to connect to auth-service", "addr", cfg.AuthServiceAddr, "error", err)
		os.Exit(1)
	}
	log.Info("connected to auth-service", "addr", cfg.AuthServiceAddr)

	// ── Audit writer ───────────────────────────────────────────────────────
	auditWriter := audit.NewWriter(db, log)

	// ── Service registry ───────────────────────────────────────────────────
	// RedisRegistry for dynamic discovery; StaticRegistry as fallback.
	redisReg := registry.NewRedisRegistry(rdb, log)
	staticReg := registry.NewStaticRegistry(map[string]string{
		"auth-service":               envOrDefault("AUTH_SERVICE_HTTP_URL", "http://localhost:8080"),
		"product-catalog-service":    envOrDefault("CATALOG_SERVICE_HTTP_URL", "http://localhost:8081"),
		"order-management-service":   envOrDefault("ORDER_SERVICE_HTTP_URL", "http://localhost:8084"),
		"currency-service":           envOrDefault("CURRENCY_SERVICE_HTTP_URL", "http://localhost:8086"),
	})

	// resolve picks an upstream address: Redis first, static fallback.
	resolve := func(ctx context.Context, name string) (string, bool) {
		if addr, ok := redisReg.Pick(ctx, name); ok {
			return addr, true
		}
		insts, _ := staticReg.Instances(ctx, name)
		if len(insts) > 0 {
			return insts[0].Addr, true
		}
		return "", false
	}

	// ── Route loader ───────────────────────────────────────────────────────
	dsn := routes.BuildDSN()
	loader := routes.NewLoader(db, dsn, log)
	if err := loader.Load(context.Background()); err != nil {
		log.Error("initial route load failed", "error", err)
		os.Exit(1)
	}

	// ── Rate limiter ───────────────────────────────────────────────────────
	rl := gw.NewRateLimiter(rdb)

	// ── Metrics tracker ────────────────────────────────────────────────────
	tracker := metrics.NewTracker()

	// ── Upstream pool ──────────────────────────────────────────────────────
	upstreams := make(map[string]*proxy.Upstream) // service name → proxy pool

	getUpstream := func(name string) *proxy.Upstream {
		if u, ok := upstreams[name]; ok {
			return u
		}
		u := proxy.New(name, log)
		u.OnRequest = func(upstream string, status int, _ time.Duration) {
			tracker.Track(upstream, status)
		}
		upstreams[name] = u
		return u
	}

	// ── Router builder (called on every route reload) ──────────────────────
	adminUIOrigin := envOrDefault("ADMIN_UI_ORIGIN", "http://localhost:3001")
	extraOrigins := strings.Split(envOrDefault("EXTRA_ALLOWED_ORIGINS", ""), ",")

	buildRouter := func() http.Handler {
		r := chi.NewRouter()
		r.Use(chimw.Recoverer)
		r.Use(gw.RequestID)
		r.Use(corsMiddleware(adminUIOrigin, cfg.AppEnv, extraOrigins))
		r.Use(gw.Blocklist(rdb))
		r.Use(rl.Limit)
		r.Use(auditMiddleware(auditWriter))

		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
		})

		// Admin API — requires admin JWT.
		adminHandler := admin.NewHandler(db, redisReg, tracker, rdb, resolve)
		r.Route("/gateway/v1", func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(admin.RequireAdmin)
			adminHandler.Mount(r)
		})

		for _, route := range loader.Routes() {
			route := route // avoid closure capture of loop variable
			upstream := getUpstream(route.Upstream)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				addr, ok := resolve(r.Context(), route.Upstream)
				if !ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte(`{"success":false,"error":{"code":"NO_UPSTREAM","message":"no healthy instances available"}}`))
					auditWriter.Log(audit.Entry{
						RequestID: gw.GetRequestID(r.Context()),
						IP:        r.RemoteAddr,
						Method:    r.Method,
						Path:      r.URL.Path,
						Upstream:  route.Upstream,
						Event:     audit.EventUpstream5xx,
						Detail:    "no healthy instances",
					})
					return
				}
				// Strip the path prefix before forwarding if configured.
				if route.StripPrefix {
					r2 := r.Clone(r.Context())
					r2.URL.Path = strings.TrimPrefix(r.URL.Path, route.PathPrefix)
					if r2.URL.Path == "" {
						r2.URL.Path = "/"
					}
					r = r2
				}
				upstream.ServeHTTP(w, r, addr)
			})

			switch route.AuthMode {
			case "none":
				r.Handle(route.PathPrefix+"/*", handler)
				r.Handle(route.PathPrefix, handler)
			case "required":
				r.With(authMW.Authenticate).Handle(route.PathPrefix+"/*", handler)
				r.With(authMW.Authenticate).Handle(route.PathPrefix, handler)
			case "method_split":
				// GET/HEAD are public; all other methods require auth.
				split := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodGet || req.Method == http.MethodHead {
						handler.ServeHTTP(w, req)
						return
					}
					authMW.Authenticate(handler).ServeHTTP(w, req)
				})
				r.Handle(route.PathPrefix+"/*", split)
				r.Handle(route.PathPrefix, split)
			}
		}

		return r
	}

	// ── Atomic router swap ─────────────────────────────────────────────────
	var routerPtr atomic.Pointer[http.Handler]
	initial := buildRouter()
	routerPtr.Store(&initial)

	// Watch for route changes and swap the router atomically.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		var lastVer int64
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if v := loader.Version(); v != lastVer {
					lastVer = v
					r := buildRouter()
					routerPtr.Store(&r)
					log.Info("router hot-reloaded", "version", v)
				}
			}
		}
	}()

	// Start route watcher and audit writer.
	loader.Watch(ctx)
	go auditWriter.Run(ctx)

	// Auto-bind watcher (opt-in).
	if os.Getenv("GATEWAY_AUTO_BIND") == "true" || cfg.AppEnv == "development" {
		binder := registry.NewAutoBinder(rdb, db, auditWriter, log)
		go binder.Run(ctx)
		log.Info("auto-bind watcher started")
	}

	// ── HTTP server with dynamic dispatch ─────────────────────────────────
	port := cfg.HTTPPort
	if port == 0 {
		port = 8000
	}

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			(*routerPtr.Load()).ServeHTTP(w, r)
		}),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting API gateway", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("gateway server error", "error", err)
		}
	}()

	<-quit
	log.Info("shutting down gateway")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	// 1. Stop accepting new HTTP requests and drain in-flight.
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("gateway shutdown error", "error", err)
	}
	// 2. Cancel context to stop background goroutines (route watcher, audit writer, auto-bind).
	cancel()
	// 3. Close auth gRPC connection after HTTP drains so in-flight auth RPCs finish.
	if err := authMW.Close(); err != nil {
		log.Error("auth middleware close error", "error", err)
	}
	// 4. Close Redis (defer in main() handles this, but explicit for clarity in order).
	// 5. Close DB (defer in main() handles this).
	log.Info("gateway stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func corsMiddleware(allowedOrigin, appEnv string, extra []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if originAllowed(origin, allowedOrigin, appEnv, extra) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Idempotency-Key")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed returns true when the request origin matches the configured
// allowed origin. In development the wildcard also covers any localhost or
// 127.0.0.1 port so multiple local frontends work without config changes.
// In all other environments only the exact allowed origin is accepted —
// the localhost wildcard would let any local process make credentialed
// cross-origin requests through a victim's browser.
// statusRecorder wraps ResponseWriter to capture the written status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// auditMiddleware emits audit events for notable response status codes.
func auditMiddleware(w *audit.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Skip health checks and OPTIONS pre-flight to keep audit log clean.
			if r.URL.Path == "/health" || r.Method == http.MethodOptions {
				next.ServeHTTP(rw, r)
				return
			}

			rec := &statusRecorder{ResponseWriter: rw, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			var event string
			switch {
			case rec.status == http.StatusUnauthorized:
				event = audit.EventAuthRejected
			case rec.status == http.StatusTooManyRequests:
				event = audit.EventRateLimited
			case rec.status >= 500:
				event = audit.EventUpstream5xx
			}
			if event == "" {
				return
			}

			userID := ""
			if u := gw.UserFromContext(r.Context()); u != nil {
				userID = u.ID
			}
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.RemoteAddr
			}
			w.Log(audit.Entry{
				RequestID:  gw.GetRequestID(r.Context()),
				UserID:     userID,
				IP:         ip,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: rec.status,
				Event:      event,
			})
		})
	}
}

func originAllowed(origin, allowed, appEnv string, extra []string) bool {
	if origin == allowed {
		return true
	}
	for _, e := range extra {
		if e = strings.TrimSpace(e); e != "" && origin == e {
			return true
		}
	}
	if appEnv == "development" {
		return strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "http://127.0.0.1:")
	}
	return false
}
