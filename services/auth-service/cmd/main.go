package main

// @title ZapMarket Auth Service API
// @version 1.0
// @description This is the authentication and user management service for ZapMarket.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /v1/auth
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

import (
	"context"
	"database/sql"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/services/auth-service/authpb"
	grpcHandler "github.com/zapmarket/zapmarket/services/auth-service/internal/handler/grpc"
	httphandler "github.com/zapmarket/zapmarket/services/auth-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	db, err := connectDB(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("Connected to database", "host", cfg.DBHost, "port", cfg.DBPort, "database", cfg.DBName)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	oauthRepo := repository.NewOAuthRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, oauthRepo, tokenRepo, cfg)
	oauthService := service.NewOAuthService(userRepo, oauthRepo, tokenRepo, cfg)

	// Initialize HTTP handlers
	httpHandler := httphandler.NewHandler(authService, oauthService, cfg)

	// Setup HTTP server
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/auth/register", httpHandler.LoggingMiddleware(httpHandler.Register))
	mux.HandleFunc("/v1/auth/login", httpHandler.LoggingMiddleware(httpHandler.Login))
	mux.HandleFunc("/v1/auth/refresh", httpHandler.LoggingMiddleware(httpHandler.Refresh))
	mux.HandleFunc("/v1/auth/me", httpHandler.LoggingMiddleware(httpHandler.Me))
	mux.HandleFunc("/v1/auth/oauth/google/url", httpHandler.LoggingMiddleware(httpHandler.GoogleOAuthURL))
	mux.HandleFunc("/v1/auth/oauth/google/callback", httpHandler.LoggingMiddleware(httpHandler.GoogleOAuthCallback))
	mux.HandleFunc("/v1/auth/oauth/facebook/url", httpHandler.LoggingMiddleware(httpHandler.FacebookOAuthURL))
	mux.HandleFunc("/v1/auth/oauth/facebook/callback", httpHandler.LoggingMiddleware(httpHandler.FacebookOAuthCallback))

	// Serve Swagger JSON
	mux.HandleFunc("/v1/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.json")
	})

	// Serve Swagger UI
	mux.Handle("/v1/docs/", httpSwagger.Handler(httpSwagger.URL("/v1/docs/swagger.json")))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	// Initialize gRPC server

	// Create channels for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		slog.Info("Starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Start gRPC server in a goroutine
	go func() {
		slog.Info("gRPC server configured", "port", cfg.GRPCPort)
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			slog.Error("Failed to listen on gRPC port", "error", err)
			sigChan <- syscall.SIGTERM
			return
		}
		grpcSrv := grpc.NewServer()
		grpcServer := grpcHandler.NewAuthServer(authService, cfg)
		authpb.RegisterAuthServiceServer(grpcSrv, grpcServer)
		reflection.Register(grpcSrv)
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcSrv.Serve(listener); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	slog.Info("Shutdown signal received, gracefully shutting down")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("Auth service stopped")
}

// connectDB establishes a PostgreSQL connection
func connectDB(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func init() {
	// Configure structured logging
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
	slog.SetDefault(logger)
}
