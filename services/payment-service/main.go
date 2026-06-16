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

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/grpcx"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	pb "github.com/zapmarket/zapmarket/pkg/proto/payment"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/gateway"
	grpchandler "github.com/zapmarket/zapmarket/services/payment-service/internal/handler/grpc"
	httphandler "github.com/zapmarket/zapmarket/services/payment-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/service"
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

	// ── Repository / Gateway / Service / gRPC handler ───────────────────────────
	repo := repository.NewPaymentRepository(db)
	paymentGateway := gateway.NewFakePaymentGateway()
	svc := service.NewPaymentService(repo, paymentGateway, log)
	grpcHandler := grpchandler.NewPaymentGRPCHandler(svc)
	webhookHandler := httphandler.NewWebhookHandler(svc, cfg.PaymentWebhookSecret, log)

	// ── HTTP (health + gateway webhook only — no public REST API) ──────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/health", webhookHandler.Health)
	mux.HandleFunc("/webhooks/payment", webhookHandler.HandlePaymentWebhook)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── gRPC server ──────────────────────────────────────────────────────────
	grpcServer := grpcx.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	// ── Start servers ────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

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
	log.Info("shutting down servers")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}
	grpcServer.GracefulStop()

	log.Info("servers stopped")
}
