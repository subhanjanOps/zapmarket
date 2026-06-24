package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	nethhttp "net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc/reflection"

	svcconfig "github.com/zapmarket/zapmarket/services/currency-service/pkg/config"

	"github.com/zapmarket/zapmarket/services/currency-service/application/ports"
	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/infrastructure/external"
	inframetrics "github.com/zapmarket/zapmarket/services/currency-service/infrastructure/metrics"
	infrapostgres "github.com/zapmarket/zapmarket/services/currency-service/infrastructure/postgres"
	infraredis "github.com/zapmarket/zapmarket/services/currency-service/infrastructure/redis"
	kafkainfra "github.com/zapmarket/zapmarket/services/currency-service/infrastructure/kafka"
	grpcserver "github.com/zapmarket/zapmarket/services/currency-service/interfaces/grpc"
	httphandler "github.com/zapmarket/zapmarket/services/currency-service/interfaces/http"
	"github.com/zapmarket/zapmarket/services/currency-service/interfaces/worker"

	"github.com/zapmarket/zapmarket/pkg/grpcx"
	"github.com/zapmarket/zapmarket/pkg/logger"
	pkgregistry "github.com/zapmarket/zapmarket/pkg/registry"
)

func main() {
	_ = godotenv.Load()

	cfg, err := svcconfig.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := openDB(cfg)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to database", "db", cfg.DBName)

	// ── Migrations ────────────────────────────────────────────────────────────
	if cfg.MigrateOnBoot {
		if err := runMigrations(cfg); err != nil {
			log.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		log.Info("migrations applied")
	}

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("Redis unreachable", "addr", cfg.RedisURL, "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to Redis", "addr", cfg.RedisURL)

	// ── Metrics ───────────────────────────────────────────────────────────────
	m := inframetrics.New()

	// ── Repositories ──────────────────────────────────────────────────────────
	currencyRepo := infrapostgres.NewCurrencyRepository(db)
	ratesRepo := infrapostgres.NewRatesRepository(db)
	ratesCache := infraredis.NewRatesCache(rdb)

	// ── FX Providers ──────────────────────────────────────────────────────────
	primaryProvider := external.NewFrankfurterProvider(cfg.RateProviderURL)
	secondaryProvider := external.NewOpenExchangeRatesProvider(cfg.FallbackRateProviderURL)
	provider := external.NewFallbackProvider(primaryProvider, secondaryProvider)

	// ── Kafka publisher (optional — no-op when KAFKA_BROKERS not set) ─────────
	var publisher ports.EventPublisher = &kafkainfra.NoopPublisher{}
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		kp := kafkainfra.NewKafkaPublisher(strings.Split(brokers, ","), "currency.rates.updated")
		defer kp.Close()
		publisher = kp
	}

	// ── Use cases ─────────────────────────────────────────────────────────────
	listCurrenciesUC := usecases.NewListCurrenciesUseCase(currencyRepo)
	getRatesUC := usecases.NewGetRatesUseCase(ratesRepo, ratesCache, cfg.RateRefreshInterval, cfg.MaxRateAge)
	ingestRatesUC := usecases.NewIngestRatesUseCase(provider, ratesRepo, ratesCache, cfg.RateRefreshInterval, log, publisher)
	toggleCurrencyUC := usecases.NewToggleCurrencyUseCase(currencyRepo)
	getRatesHistoryUC := usecases.NewGetRatesHistoryUseCase(ratesRepo)

	// ── HTTP server ───────────────────────────────────────────────────────────
	httpHandler := httphandler.NewHandler(listCurrenciesUC, getRatesUC, toggleCurrencyUC, getRatesHistoryUC, log)

	authMW, err := httphandler.NewAuthMiddleware(cfg.AuthServiceAddr, log)
	if err != nil {
		log.Error("failed to dial auth-service", "error", err)
		os.Exit(1)
	}
	defer authMW.Close()

	liveness := func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("db: %w", err)
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		return nil
	}

	router := httphandler.NewRouter(httpHandler, m, liveness, authMW)

	httpServer := &nethhttp.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── gRPC server ───────────────────────────────────────────────────────────
	grpcSrv := grpcx.NewServer()
	currencyGRPCServer := grpcserver.NewCurrencyServer(listCurrenciesUC, getRatesUC, log)
	currencyGRPCServer.Register(grpcSrv)
	reflection.Register(grpcSrv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Worker ────────────────────────────────────────────────────────────────
	ingestor := worker.NewRateIngestor(ingestRatesUC, cfg.RateRefreshInterval, "USD", m, log)
	go ingestor.Run(ctx)

	// ── Signals ───────────────────────────────────────────────────────────────
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ── Start HTTP ────────────────────────────────────────────────────────────
	go func() {
		log.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, nethhttp.ErrServerClosed) {
			log.Error("HTTP server error", "error", err)
		}
	}()

	// ── Start gRPC ────────────────────────────────────────────────────────────
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			log.Error("failed to listen on gRPC port", "port", cfg.GRPCPort, "error", err)
			sigChan <- syscall.SIGTERM
			return
		}
		log.Info("starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	// ── Registry heartbeat ────────────────────────────────────────────────────
	instanceID := uuid.New().String()
	addr := fmt.Sprintf("http://zapmarket-currency-service:%d", cfg.HTTPPort)
	go pkgregistry.Heartbeat(ctx, rdb, cfg.ServiceName, instanceID, addr, log)
	log.Info("registered with gateway registry", "addr", addr)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	<-sigChan
	log.Info("shutdown signal received")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()

	if err := httpServer.Shutdown(shutCtx); err != nil {
		log.Error("HTTP shutdown error", "error", err)
	}
	grpcSrv.GracefulStop()
	log.Info("currency-service stopped")
}

func openDB(cfg *svcconfig.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func runMigrations(cfg *svcconfig.Config) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func init() {
	slog.SetDefault(logger.New(os.Getenv("APP_ENV")))
}
