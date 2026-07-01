package main

import (
	"context"
	"database/sql"
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
	"github.com/zapmarket/zapmarket/pkg/crypto"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	httphandler "github.com/zapmarket/zapmarket/services/review-return-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/review-return-service/internal/infrastructure/postgres"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("services/review-return-service/.env")
	}

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
		log.Info("migrations applied")
	}

	repo := postgres.New(db)
	h := httphandler.New(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	requireAuth := crypto.RequireAuth(cfg.JWTSecretKey)
	requireStaff := crypto.RequireRole("seller", "admin")

	mux.Handle("POST /v1/returns", requireAuth(http.HandlerFunc(h.CreateReturn)))
	mux.Handle("PUT /v1/returns/{id}/approve", requireAuth(requireStaff(http.HandlerFunc(h.ApproveReturn))))
	mux.Handle("PUT /v1/returns/{id}/reject", requireAuth(requireStaff(http.HandlerFunc(h.RejectReturn))))
	mux.HandleFunc("GET /v1/products/{id}/rating", h.GetProductRating)

	port := cfg.HTTPPort
	if port == 0 {
		port = 8093
	}
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go runRatingRefresher(ctx, db, log)

	go func() {
		log.Info("review-return-service listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
		}
	}()

	<-quit
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)
	log.Info("review-return-service stopped")
}

func runRatingRefresher(ctx context.Context, db *sql.DB, log *slog.Logger) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := db.ExecContext(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY product_ratings`); err != nil {
				log.Warn("failed to refresh product_ratings", "error", err)
			} else {
				log.Info("refreshed product_ratings")
			}
		}
	}
}
