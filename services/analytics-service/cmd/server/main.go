package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	pkgcfg "github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/services/analytics-service/internal/handler"
)

func main() {
	_ = godotenv.Load()

	cfg, err := pkgcfg.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	db, err := database.New(cfg)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if cfg.MigrateOnBoot {
		if err := migrate.Up(cfg, "migrations"); err != nil {
			slog.Error("migrations failed", "error", err)
			os.Exit(1)
		}
	}

	logger := slog.Default()
	run(db, cfg, logger)
}

func run(db *sql.DB, cfg *pkgcfg.Config, logger *slog.Logger) {
	writer := handler.NewEventWriter(db, logger, 1000, 4)
	h := handler.NewEventHandler(writer, logger)

	mux := http.NewServeMux()
	mux.Handle("POST /v1/events", crypto.OptionalAuth(cfg.JWTSecretKey)(http.HandlerFunc(h.RecordEvent)))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	port := cfg.HTTPPort
	if port == 0 {
		port = 8095
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("analytics-service starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
		}
	}()

	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	writer.Close(ctx)
	slog.Info("analytics-service shut down")
}
