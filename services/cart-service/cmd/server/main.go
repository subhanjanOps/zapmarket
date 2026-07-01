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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	cataloggrpc "github.com/zapmarket/zapmarket/services/cart-service/internal/infrastructure/grpc"
	cartpostgres "github.com/zapmarket/zapmarket/services/cart-service/internal/infrastructure/postgres"
	cartredis "github.com/zapmarket/zapmarket/services/cart-service/internal/infrastructure/redis"
	carthttp "github.com/zapmarket/zapmarket/services/cart-service/internal/interfaces/http"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/application/usecases"
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

	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	catalogAddr := os.Getenv("CATALOG_GRPC_ADDR")
	if catalogAddr == "" {
		catalogAddr = "localhost:50052"
	}
	conn, err := grpc.NewClient(catalogAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to dial catalog service", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	skuFetcher := cataloggrpc.NewCatalogSKUFetcher(conn)
	authRepo := cartpostgres.NewCartRepo(db)
	guestRepo := cartredis.NewGuestCartRepo(rdb)

	addUC := usecases.NewAddItemUseCase(authRepo, skuFetcher)
	removeUC := usecases.NewRemoveItemUseCase(authRepo)
	getUC := usecases.NewGetCartUseCase(authRepo)
	mergeUC := usecases.NewMergeCartUseCase(guestRepo, authRepo, skuFetcher)

	handler := carthttp.NewCartHandler(addUC, removeUC, getUC, mergeUC)

	r := chi.NewRouter()
	r.Mount("/v1/cart", handler.Routes(cfg.JWTSecretKey))
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
		log.Info("cart-service started", "port", cfg.HTTPPort)
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
	log.Info("cart-service stopped")
}
