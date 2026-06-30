package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/pkg/relay"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/carrier/shiprocket"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/carrier/stub"
	loghandler "github.com/zapmarket/zapmarket/services/logistics-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/infrastructure/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("services/logistics-service/.env")
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
	}

	agents := repository.NewAgentRepository(db)
	shipments := repository.NewShipmentRepository(db)
	pods := repository.NewPODRepository(db)
	tracking := repository.NewTrackingRepository(db)

	// Carrier selection via CARRIER_PROVIDER env var (default: stub)
	type carrierIface interface {
		CreateReversePickup(ctx context.Context, parentAWB, orderID string) (awb, carrier, trackingURL string, err error)
	}
	var carrierClient carrierIface
	if os.Getenv("CARRIER_PROVIDER") == "shiprocket" {
		carrierClient = shiprocket.New(
			os.Getenv("SHIPROCKET_EMAIL"),
			os.Getenv("SHIPROCKET_PASSWORD"),
			os.Getenv("SHIPROCKET_CHANNEL_ID"),
		)
		log.Info("carrier: Shiprocket")
	} else {
		carrierClient = stub.New()
		log.Info("carrier: stub")
	}

	h := loghandler.NewHandler(agents, shipments, pods, tracking, carrierClient)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("POST /v1/agents", h.CreateAgent)
	mux.HandleFunc("GET /v1/agents", h.ListAgents)
	mux.HandleFunc("PUT /v1/shipments/{id}/assign", h.AssignAgent)
	mux.HandleFunc("POST /v1/shipments/{id}/attempt", h.RecordAttempt)
	mux.HandleFunc("POST /v1/shipments/{id}/deliver", h.Deliver)
	mux.HandleFunc("GET /v1/shipments/{id}/tracking", h.GetTracking)
	mux.HandleFunc("POST /v1/webhooks/shiprocket", h.ShiprocketWebhook)
	mux.HandleFunc("POST /v1/return-shipments", h.CreateReturnShipment)

	port := cfg.HTTPPort
	if port == 0 {
		port = 8092
	}
	addr := fmt.Sprintf(":%d", port)

	shipmentProducer := pkgkafka.NewProducer(cfg.KafkaBrokers, pkgkafka.TopicShipmentDelivered)
	outboxRelay := relay.New(db, shipmentProducer, pkgkafka.TopicShipmentDelivered, log)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	relayCtx, relayCancel := context.WithCancel(context.Background())
	go outboxRelay.Run(relayCtx)
	log.Info("logistics-service starting", "addr", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
		}
	}()

	<-quit
	relayCancel()
	_ = shipmentProducer.Close()
	log.Info("logistics-service stopped")
}
