package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc/reflection"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/grpcx"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	pb "github.com/zapmarket/zapmarket/pkg/proto/inventory"
	pkgmetrics "github.com/zapmarket/zapmarket/pkg/metrics"
	grpchandler "github.com/zapmarket/zapmarket/services/inventory-service/internal/handler/grpc"
	httphandler "github.com/zapmarket/zapmarket/services/inventory-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/infrastructure/cache"
	"github.com/zapmarket/zapmarket/pkg/relay"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/service"
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
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to Redis", "addr", cfg.RedisURL, "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to Redis", "addr", cfg.RedisURL)

	// ── Metrics ───────────────────────────────────────────────────────────────
	m := pkgmetrics.New("inventory")

	// ── Repository / Service / gRPC handler ─────────────────────────────────────
	repo := repository.NewInventoryRepository(db)
	stockCache := cache.NewRedisStockCache(rdb)
	svc := service.NewInventoryService(repo, stockCache, log)
	grpcHandler := grpchandler.NewInventoryGRPCHandler(svc)

	// ── HTTP (health check only — no public REST API, see design.md) ───────────
	mux := http.NewServeMux()
	mux.HandleFunc("/health", httphandler.Health)
	mux.Handle("/metrics", m.Handler())

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      m.Middleware()(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── gRPC server ──────────────────────────────────────────────────────────
	grpcServer := grpcx.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	// ── Outbox relay ─────────────────────────────────────────────────────────
	inventoryProducer := pkgkafka.NewProducer(cfg.KafkaBrokers, pkgkafka.TopicInventory)
	outboxRelay := relay.New(db, inventoryProducer, pkgkafka.TopicInventory, log)

	// ── Start servers ────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	relayCtx, relayCancel := context.WithCancel(context.Background())
	go outboxRelay.Run(relayCtx)
	log.Info("outbox relay started", "brokers", cfg.KafkaBrokers)

	go func() {
		log.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	go func() {
		log.Info("starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	<-quit
	relayCancel()
	log.Info("shutting down servers")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}
	grpcServer.GracefulStop()
	_ = inventoryProducer.Close()

	log.Info("servers stopped")
}
