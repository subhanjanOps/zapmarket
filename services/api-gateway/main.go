package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/logger"
	gw "github.com/zapmarket/zapmarket/services/api-gateway/internal/middleware"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/proxy"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)

	// ── Redis ──────────────────────────────────────────────────────────────
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to Redis", "addr", cfg.RedisURL, "error", err)
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

	rl := gw.NewRateLimiter(rdb)

	// ── Upstream proxies ───────────────────────────────────────────────────
	authURL := envOrDefault("AUTH_SERVICE_HTTP_URL", "http://localhost:8080")
	catalogURL := envOrDefault("CATALOG_SERVICE_HTTP_URL", "http://localhost:8081")
	orderURL := envOrDefault("ORDER_SERVICE_HTTP_URL", "http://localhost:8084")

	authProxy, err := proxy.New("auth-service", authURL, log)
	if err != nil {
		log.Error("failed to create auth proxy", "error", err)
		os.Exit(1)
	}
	catalogProxy, err := proxy.New("product-catalog-service", catalogURL, log)
	if err != nil {
		log.Error("failed to create catalog proxy", "error", err)
		os.Exit(1)
	}
	orderProxy, err := proxy.New("order-management-service", orderURL, log)
	if err != nil {
		log.Error("failed to create order proxy", "error", err)
		os.Exit(1)
	}

	// ── Router ────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(gw.RequestID)
	r.Use(rl.Limit) // rate limit applies to all routes

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
	})

	// ── Auth routes ────────────────────────────────────────────────────────
	// Public (no JWT required): register, login, refresh, OAuth.
	r.Route("/v1/auth", func(r chi.Router) {
		r.Handle("/register", authProxy)
		r.Handle("/login", authProxy)
		r.Handle("/refresh", authProxy)
		r.Handle("/oauth/*", authProxy)

		// Protected auth routes (need valid JWT).
		r.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Handle("/me", authProxy)
			r.Handle("/logout", authProxy)
		})
	})

	// ── Catalog routes — public reads, protected writes ────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public reads: product list, single product, categories, SKUs.
		r.Handle("/categories", catalogProxy)
		r.Handle("/categories/*", catalogProxy)
		r.Handle("/products", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				catalogProxy.ServeHTTP(w, r)
				return
			}
			// POST/PUT/DELETE require auth.
			authMW.Authenticate(catalogProxy).ServeHTTP(w, r)
		}))
		r.Handle("/products/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				catalogProxy.ServeHTTP(w, r)
				return
			}
			authMW.Authenticate(catalogProxy).ServeHTTP(w, r)
		}))
	})

	// ── Order routes — all protected ───────────────────────────────────────
	r.Route("/v1/orders", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Handle("/*", stripPrefix("/v1/orders", orderProxy))
		r.Handle("/", orderProxy)
	})

	port := cfg.HTTPPort
	if port == 0 {
		port = 8000
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting API gateway", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("gateway error", "error", err)
		}
	}()

	<-quit
	log.Info("shutting down gateway")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("gateway shutdown error", "error", err)
	}
	log.Info("gateway stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func stripPrefix(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}
		h.ServeHTTP(w, r2)
	})
}
