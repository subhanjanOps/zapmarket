package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application/usecases"
	kafkaconsumer "github.com/zapmarket/zapmarket/services/settlement-service/internal/infrastructure/kafka"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/infrastructure/postgres"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/infrastructure/razorpay"
	settlementhttp "github.com/zapmarket/zapmarket/services/settlement-service/internal/interfaces/http"
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

	commissionBPS := int64(200)
	if v := os.Getenv("PLATFORM_COMMISSION_BPS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			commissionBPS = n
		}
	}
	minPayoutPaise := int64(10000)
	if v := os.Getenv("MIN_PAYOUT_PAISE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			minPayoutPaise = n
		}
	}

	ledgerRepo := postgres.NewLedgerRepo(db)
	gateway := razorpay.NewNoopGateway(log)

	creditUC := usecases.NewCreditSaleUseCase(ledgerRepo, commissionBPS)
	debitUC := usecases.NewDebitRefundUseCase(ledgerRepo)
	payoutUC := usecases.NewInitiatePayoutUseCase(ledgerRepo, gateway, minPayoutPaise)
	_ = payoutUC // wired to weekly scheduler in production

	consumer := kafkaconsumer.NewPaymentConsumer(creditUC, debitUC, log)

	capturedConsumer := pkgkafka.NewConsumer(cfg.KafkaBrokers, pkgkafka.TopicPaymentCaptured, "settlement-captured")

	balanceH := settlementhttp.NewBalanceHandler(ledgerRepo)

	r := chi.NewRouter()
	r.Get("/v1/sellers/{id}/balance", balanceH.GetBalance)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := capturedConsumer.Run(ctx, consumer.HandleCaptured); err != nil {
			log.Error("payment captured consumer exited", "error", err)
		}
	}()
	log.Info("settlement consumers started")

	go func() {
		log.Info("settlement-service started", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	<-quit
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("shutdown error", "error", err)
	}
	_ = capturedConsumer.Close()
	log.Info("settlement-service stopped")
}
