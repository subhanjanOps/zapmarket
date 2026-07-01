package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/wishlist-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/wishlist-service/internal/infrastructure/postgres"
	wishlisthttp "github.com/zapmarket/zapmarket/services/wishlist-service/internal/interfaces/http"
)

const maxWishlistItems = 200

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
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

	repo := postgres.NewWishlistRepo(db)
	addUC := usecases.NewAddToWishlistUseCase(repo, maxWishlistItems)
	listUC := usecases.NewListWishlistUseCase(repo)
	removeUC := usecases.NewRemoveFromWishlistUseCase(repo)
	clearUC := usecases.NewClearWishlistUseCase(repo)

	handler := wishlisthttp.NewWishlistHandler(addUC, listUC, removeUC, clearUC)

	r := chi.NewRouter()
	r.Mount("/v1/wishlist", handler.Routes(cfg.JWTSecretKey))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("wishlist-service started", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}
	log.Info("wishlist-service stopped")
}
