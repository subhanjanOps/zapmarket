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
	"github.com/zapmarket/zapmarket/pkg/crypto"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application"
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
	bankAccountRepo := postgres.NewBankAccountRepo(db)

	// Payout gateway: Razorpay if keys are configured, noop otherwise
	var payoutGateway application.PayoutGateway
	if keyID := os.Getenv("RAZORPAY_KEY_ID"); keyID != "" {
		payoutGateway = razorpay.NewPayoutGateway(keyID, os.Getenv("RAZORPAY_KEY_SECRET"), os.Getenv("RAZORPAY_ACCOUNT_NUMBER"))
		log.Info("payout gateway: Razorpay")
	} else {
		payoutGateway = razorpay.NewNoopGateway(log)
		log.Info("payout gateway: noop (set RAZORPAY_KEY_ID to enable)")
	}

	creditUC := usecases.NewCreditSaleUseCase(ledgerRepo, commissionBPS)
	debitUC := usecases.NewDebitRefundUseCase(ledgerRepo)
	payoutUC := usecases.NewInitiatePayoutUseCase(ledgerRepo, payoutGateway, minPayoutPaise)

	consumer := kafkaconsumer.NewPaymentConsumer(creditUC, debitUC, log)
	capturedConsumer := pkgkafka.NewConsumer(cfg.KafkaBrokers, pkgkafka.TopicPaymentCaptured, "settlement-captured")
	refundedConsumer := pkgkafka.NewConsumer(cfg.KafkaBrokers, pkgkafka.TopicPaymentRefunded, "settlement-refunded")

	balanceH := settlementhttp.NewBalanceHandler(ledgerRepo)
	bankH := settlementhttp.NewBankAccountHandler(bankAccountRepo)

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Group(func(pr chi.Router) {
		pr.Use(crypto.RequireAuth(cfg.JWTSecretKey))
		pr.Get("/v1/sellers/{id}/balance", balanceH.GetBalance)
		pr.Post("/v1/sellers/{id}/bank-accounts", bankH.Create)
		pr.Get("/v1/sellers/{id}/bank-accounts", bankH.List)
	})

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
	go func() {
		if err := refundedConsumer.Run(ctx, consumer.HandleRefunded); err != nil {
			log.Error("payment refunded consumer exited", "error", err)
		}
	}()
	log.Info("settlement consumers started")

	// Weekly payout scheduler: every Monday 10am
	go runWeeklyPayoutScheduler(ctx, ledgerRepo, payoutUC, log)

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
	_ = refundedConsumer.Close()
	log.Info("settlement-service stopped")
}

// runWeeklyPayoutScheduler fires InitiatePayoutUseCase for all pending sellers every Monday at 10am IST.
func runWeeklyPayoutScheduler(ctx context.Context, ledger application.LedgerRepository, payoutUC *usecases.InitiatePayoutUseCase, log *slog.Logger) {
	for {
		now := time.Now().UTC()
		daysUntilMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
		if daysUntilMonday == 0 && now.Hour() >= 4 {
			daysUntilMonday = 7
		}
		next := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 4, 30, 0, 0, time.UTC)
		delay := next.Sub(now)
		log.Info("payout scheduler: next run", "at", next.Format(time.RFC3339), "in", delay.Round(time.Minute))

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		log.Info("payout scheduler: initiating weekly payouts")
		sellers, err := ledger.GetPendingSellers(ctx, 10000)
		if err != nil {
			log.Error("payout scheduler: failed to get pending sellers", "error", err)
			continue
		}
		for _, sellerID := range sellers {
			if err := payoutUC.Execute(ctx, sellerID); err != nil {
				log.Error("payout scheduler: payout failed", "seller_id", sellerID, "error", err)
			}
		}
		log.Info("payout scheduler: completed", "sellers_processed", len(sellers))
	}
}
