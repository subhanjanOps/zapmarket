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

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/promotions-service/internal/application/usecases"
	couponhttp "github.com/zapmarket/zapmarket/services/promotions-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/promotions-service/internal/infrastructure/repository"
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

	if cfg.MigrateOnBoot {
		if err := migrate.Up(cfg, "migrations"); err != nil {
			log.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	repo := repository.NewCouponRepository(db)
	validateUC := usecases.NewValidateCouponUseCase(repo)
	handler := couponhttp.NewCouponHandler(validateUC, repo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /v1/coupons/validate", handler.Validate)
	mux.HandleFunc("POST /v1/coupons/{coupon_id}/redeem", handler.Redeem)
	mux.HandleFunc("GET /v1/promotions/active-sales", handler.GetActiveSales)

	port := cfg.HTTPPort
	if port == 0 {
		port = 8091
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("promotions-service listening", "port", fmt.Sprintf("%d", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Info("promotions-service stopped")
}
