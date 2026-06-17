package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/pkg/config"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/services/notification-service/internal/consumer"
	"github.com/zapmarket/zapmarket/services/notification-service/internal/notifier"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to Redis", "addr", cfg.RedisURL, "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to Redis", "addr", cfg.RedisURL)

	// ── Notifier ──────────────────────────────────────────────────────────────
	n := notifier.NewLogNotifier(log)

	// ── Kafka consumer ────────────────────────────────────────────────────────
	orderConsumer := pkgkafka.NewConsumer(cfg.KafkaBrokers, "orders", "notification-service")
	defer orderConsumer.Close()
	log.Info("kafka consumer ready", "brokers", cfg.KafkaBrokers, "topic", "orders")

	handler := consumer.New(n, rdb, log)

	// ── Health endpoint ───────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	go func() {
		if err := http.ListenAndServe(":8085", mux); err != nil {
			log.Error("health server error", "error", err)
		}
	}()

	// ── Signal handling ───────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-quit
		log.Info("shutting down notification service")
		cancel()
	}()

	log.Info("notification service started, consuming events")
	if err := orderConsumer.Run(ctx, handler.Handle); err != nil {
		log.Error("consumer exited with error", "error", err)
		os.Exit(1)
	}

	log.Info("notification service stopped")
}
