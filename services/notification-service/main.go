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

	"fmt"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/pkg/config"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/pkg/logger"
	pkgmetrics "github.com/zapmarket/zapmarket/pkg/metrics"
	"github.com/zapmarket/zapmarket/services/notification-service/internal/consumer"
	"github.com/zapmarket/zapmarket/services/notification-service/internal/infrastructure/cache"
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

	// ── Metrics ───────────────────────────────────────────────────────────────
	m := pkgmetrics.New("notification")

	// ── Notifier ──────────────────────────────────────────────────────────────
	// Use SMTP when credentials are configured; fall back to log-only in dev.
	var n notifier.Notifier
	if cfg.SMTPHost != "" && cfg.SMTPFrom != "" {
		n = notifier.NewSMTPNotifier(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
		log.Info("SMTP notifier active", "host", cfg.SMTPHost, "from", cfg.SMTPFrom)
	} else {
		n = notifier.NewLogNotifier(log)
		log.Info("SMTP not configured — notifications will be logged only")
	}

	dedup := cache.NewRedisDeduplicator(rdb)
	handler := consumer.New(n, dedup, log)

	// ── Health + metrics endpoint ─────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/metrics", m.Handler())
	healthSrv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: mux}
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

const (
	retryBaseDelay = 2 * time.Second
	retryMaxDelay  = 2 * time.Minute
)

// runConsumer runs a consumer for a single topic, restarting on transient errors
// with exponential backoff (2s → 4s → 8s … capped at 2 min).
func runConsumer(ctx context.Context, brokers []string, topic string, h *consumer.Handler, log *slog.Logger) {
	log.Info("starting consumer", "topic", topic)
	delay := retryBaseDelay
	for {
		if ctx.Err() != nil {
			return
		}
		c := pkgkafka.NewConsumer(brokers, topic, "notification-service")
		start := time.Now()
		if err := c.Run(ctx, h.Handle); err != nil {
			_ = c.Close()
			// Reset backoff if the consumer ran for at least one full window before failing.
			if time.Since(start) > retryMaxDelay {
				delay = retryBaseDelay
			}
			log.Error("consumer error, retrying", "topic", topic, "error", err, "backoff", delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			delay *= 2
			if delay > retryMaxDelay {
				delay = retryMaxDelay
			}
			continue
		}
		_ = c.Close()
		return
	}
}
