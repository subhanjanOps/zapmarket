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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/clients"
	httphandler "github.com/zapmarket/zapmarket/services/order-management-service/internal/handler/http"
	authmw "github.com/zapmarket/zapmarket/services/order-management-service/internal/middleware"
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

	// ── Downstream clients ────────────────────────────────────────────────────
	inventoryClient, err := clients.NewInventoryClient(cfg.InventoryServiceAddr)
	if err != nil {
		log.Error("failed to connect to inventory-service", "addr", cfg.InventoryServiceAddr, "error", err)
		os.Exit(1)
	}
	log.Info("connected to inventory-service", "addr", cfg.InventoryServiceAddr)

	paymentClient, err := clients.NewPaymentClient(cfg.PaymentServiceAddr)
	if err != nil {
		log.Error("failed to connect to payment-service", "addr", cfg.PaymentServiceAddr, "error", err)
		os.Exit(1)
	}
	log.Info("connected to payment-service", "addr", cfg.PaymentServiceAddr)

	// ── Auth middleware ───────────────────────────────────────────────────────
	authMW, err := authmw.NewAuthMiddleware(cfg.AuthServiceAddr, log)
	if err != nil {
		log.Error("failed to connect to auth-service", "addr", cfg.AuthServiceAddr, "error", err)
		os.Exit(1)
	}
	log.Info("connected to auth-service", "addr", cfg.AuthServiceAddr)

	// ── Repository / Service / Handler ───────────────────────────────────────
	repo := repository.NewOrderRepository(db)
	svc := service.NewOrderService(repo, inventoryClient, paymentClient, log)
	handler := httphandler.NewOrderHandler(svc)

	// ── Router ───────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httphandler.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/v1/orders", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(authMW.RequireRole("buyer", "seller", "admin"))
		r.Post("/", handler.Checkout)
		r.Get("/", handler.ListOrders)
		r.Get("/{id}", handler.GetOrder)
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Start / shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	<-quit
	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}

	log.Info("server stopped")
}
