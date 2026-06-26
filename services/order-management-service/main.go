// @title			Order Management Service API
// @version		1.0
// @description	Handles order checkout, retrieval, and cancellation for ZapMarket.
// @host			localhost:8082
// @BasePath		/
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter: Bearer <token>
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	pkgmetrics "github.com/zapmarket/zapmarket/pkg/metrics"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/pkg/registry"
	"github.com/zapmarket/zapmarket/pkg/swaggerx"
	_ "github.com/zapmarket/zapmarket/services/order-management-service/docs"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/clients"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
	httphandler "github.com/zapmarket/zapmarket/services/order-management-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/infrastructure/cache"
	authmw "github.com/zapmarket/zapmarket/services/order-management-service/internal/middleware"
	"github.com/zapmarket/zapmarket/pkg/relay"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/service"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)

	db, err := database.New(cfg)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to database")

	if cfg.MigrateOnBoot {
		if err := migrate.Up(cfg, "migrations"); err != nil {
			log.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		log.Info("migrations applied")
	}

	// ── Redis ─────────────────────────────────────────────────────────────────
	// Redis is used for idempotency caching only — not for correctness. The service
	// degrades to DB-only idempotency checks when Redis is unavailable.
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warn("Redis unavailable, running without cache (DB-only idempotency)", "addr", cfg.RedisURL, "error", err)
		rdb = nil
	} else {
		defer rdb.Close()
		log.Info("connected to Redis", "addr", cfg.RedisURL)
	}

	// ── Downstream clients ────────────────────────────────────────────────────
	inventoryClient, err := clients.NewInventoryClient(cfg.InventoryServiceAddr)
	if err != nil {
		log.Error("failed to connect to inventory-service", "addr", cfg.InventoryServiceAddr, "error", err)
		os.Exit(1)
	}
	defer inventoryClient.Close()
	log.Info("connected to inventory-service", "addr", cfg.InventoryServiceAddr)

	paymentClient, err := clients.NewPaymentClient(cfg.PaymentServiceAddr)
	if err != nil {
		log.Error("failed to connect to payment-service", "addr", cfg.PaymentServiceAddr, "error", err)
		os.Exit(1)
	}
	defer paymentClient.Close()
	log.Info("connected to payment-service", "addr", cfg.PaymentServiceAddr)

	// ── Auth middleware ───────────────────────────────────────────────────────
	authMW, err := authmw.NewAuthMiddleware(cfg.AuthServiceAddr, log)
	if err != nil {
		log.Error("failed to connect to auth-service", "addr", cfg.AuthServiceAddr, "error", err)
		os.Exit(1)
	}
	log.Info("connected to auth-service", "addr", cfg.AuthServiceAddr)

	// ── Metrics ───────────────────────────────────────────────────────────────
	m := pkgmetrics.New("order")

	// ── Repository / Service / Handler ───────────────────────────────────────
	repo := repository.NewOrderRepository(db)
	var orderCache contracts.OrderCache
	if rdb != nil {
		orderCache = cache.NewRedisCache(rdb)
	} else {
		orderCache = cache.NewNoopCache()
	}
	svc := service.NewOrderService(repo, inventoryClient, paymentClient, orderCache, log)
	handler := httphandler.NewOrderHandler(svc)
	adminHandler := httphandler.NewAdminOrderHandler(svc)

	// ── Router ───────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(httpx.LimitBody(httpx.MaxBodyBytes))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httphandler.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", m.Handler())

	// Admin order routes — require admin JWT role.
	// Register /v1/admin/orders BEFORE any broader /v1/admin prefix on another service.
	r.Route("/v1/admin/orders", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(authMW.RequireRole("admin"))
		r.Get("/", adminHandler.AdminListOrders)
		r.Get("/{id}", adminHandler.AdminGetOrder)
		r.Post("/{id}/cancel", adminHandler.AdminCancelOrder)
	})

	r.Route("/v1/orders", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(authMW.RequireRole("buyer", "seller", "admin"))
		r.Post("/", handler.Checkout)
		r.Get("/", handler.ListOrders)
		// Seller-scoped routes — must be registered before /{id} to avoid ambiguity.
		r.Get("/seller", handler.ListSellerOrders)
		r.Get("/seller/{id}", handler.GetSellerOrder)
		r.Get("/{id}", handler.GetOrder)
		r.Post("/{id}/cancel", handler.CancelOrder)
	})

	r.Get("/v1/docs/swagger.json", swaggerx.JSONHandler(""))
	r.Get("/v1/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/v1/docs/swagger.json"),
	))

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Outbox relay ─────────────────────────────────────────────────────────
	orderProducer := pkgkafka.NewProducer(cfg.KafkaBrokers, "orders")
	outboxRelay := relay.New(db, orderProducer, "orders", log)

	// ── Start / shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	relayCtx, relayCancel := context.WithCancel(context.Background())
	go outboxRelay.Run(relayCtx)
	log.Info("outbox relay started", "brokers", cfg.KafkaBrokers)

	instanceID := uuid.New().String()
	addr := fmt.Sprintf("http://zapmarket-order-management-service:%d", cfg.HTTPPort)
	if rdb != nil {
		go registry.Heartbeat(relayCtx, rdb, "order-management-service", instanceID, addr, log)
		log.Info("registered with gateway registry", "addr", addr)
	} else {
		log.Warn("skipping gateway registry heartbeat — Redis unavailable")
	}

	go func() {
		log.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	<-quit
	relayCancel()
	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}
	_ = orderProducer.Close()

	log.Info("server stopped")
}
