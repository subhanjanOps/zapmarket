package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	handler := consumer.New(n, rdb, log)

	// ── Health endpoint ───────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	healthSrv := &http.Server{Addr: ":8085", Handler: mux}
	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		if err := healthSrv.Shutdown(shutCtx); err != nil {
			log.Error("health server shutdown error", "error", err)
		}
	}()

	// ── Consume all three topics concurrently ─────────────────────────────────
	topics := []string{
		pkgkafka.TopicOrders,
		pkgkafka.TopicPayments,
		pkgkafka.TopicInventory,
	}

	var wg sync.WaitGroup
	for _, topic := range topics {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			runConsumer(ctx, cfg.KafkaBrokers, t, handler, log)
		}(topic)
	}

	log.Info("notification service started", "topics", topics)
	wg.Wait()
	log.Info("notification service stopped")
}

// runConsumer runs a consumer for a single topic, restarting on transient errors.
func runConsumer(ctx context.Context, brokers []string, topic string, h *consumer.Handler, log *slog.Logger) {
	log.Info("starting consumer", "topic", topic)
	for {
		if ctx.Err() != nil {
			return
		}
		c := pkgkafka.NewConsumer(brokers, topic, "notification-service")
		if err := c.Run(ctx, h.Handle); err != nil {
			_ = c.Close()
			log.Error("consumer error, retrying in 5s", "topic", topic, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		_ = c.Close()
		return
	}
}
