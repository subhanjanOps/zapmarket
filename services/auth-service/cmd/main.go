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
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"


	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc/reflection"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/pkg/telemetry"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/grpcx"
	"github.com/zapmarket/zapmarket/pkg/logger"
	pkgmetrics "github.com/zapmarket/zapmarket/pkg/metrics"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	"github.com/zapmarket/zapmarket/pkg/registry"
	authpb "github.com/zapmarket/zapmarket/pkg/proto/auth"
	"github.com/zapmarket/zapmarket/pkg/swaggerx"
	_ "github.com/zapmarket/zapmarket/services/auth-service/docs"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/email"
	grpcHandler "github.com/zapmarket/zapmarket/services/auth-service/internal/handler/grpc"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
	httphandler "github.com/zapmarket/zapmarket/services/auth-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/infrastructure/redisstore"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/service"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/sms"
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
	db, err := database.New(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("Connected to database", "host", cfg.DBHost, "port", cfg.DBPort, "database", cfg.DBName)

	if cfg.MigrateOnBoot {
		if err := migrate.Up(cfg, "migrations"); err != nil {
			slog.Error("Failed to run migrations", "error", err)
			os.Exit(1)
		}
		slog.Info("Migrations applied")
	}

	// ── Telemetry ─────────────────────────────────────────────────────────────
	shutdownTracing, err := telemetry.Setup(context.Background(), "auth-service", cfg.OTLPEndpoint)
	if err != nil {
		slog.Warn("tracing unavailable", "error", err)
	} else {
		defer func() { _ = shutdownTracing(context.Background()) }()
	}

	// ── Metrics ───────────────────────────────────────────────────────────────
	m := pkgmetrics.New("auth")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	oauthRepo := repository.NewOAuthRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	resetRepo := repository.NewPasswordResetRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	prefsRepo := repository.NewPreferencesRepository(db)
	sellerProfileRepo := repository.NewSellerProfileRepository(db)

	// ── Redis (optional — auth still works without it) ───────────────────────
	var rdb *goredis.Client
	redisClient := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Redis unavailable — token blacklist and registry disabled", "addr", cfg.RedisURL, "error", err)
		_ = redisClient.Close()
	} else {
		rdb = redisClient
		defer rdb.Close()
		slog.Info("connected to Redis", "addr", cfg.RedisURL)
	}

	// Choose emailer: Resend > SMTP > LogEmailer (dev fallback).
	var emailer email.Emailer
	switch {
	case cfg.ResendAPIKey != "":
		emailer = email.NewResendEmailer(cfg.ResendAPIKey, cfg.EmailFrom)
		slog.Info("using Resend emailer", "from", cfg.EmailFrom)
	case cfg.SMTPHost != "" && cfg.SMTPUser != "":
		emailer = email.NewSMTPEmailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
		slog.Info("using SMTP emailer", "host", cfg.SMTPHost)
	default:
		emailer = email.NewLogEmailer()
		slog.Info("using log emailer (set RESEND_API_KEY or SMTP_HOST to enable email)")
	}

	// Wire Redis-backed or no-op stores depending on Redis availability.
	var blacklist contracts.TokenBlacklist
	var oauthStateStore contracts.OAuthStateStore
	if rdb != nil {
		blacklist = redisstore.NewTokenBlacklist(rdb)
		oauthStateStore = redisstore.NewOAuthStateStore(rdb)
	} else {
		blacklist = &redisstore.NoopTokenBlacklist{}
		oauthStateStore = &redisstore.NoopOAuthStateStore{}
	}

	// Wire SMS sender: Twilio if configured, else log-only.
	var smser sms.SMSer
	if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" {
		smser = sms.NewTwilioSMSer(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber)
		slog.Info("using Twilio SMS sender")
	} else {
		smser = sms.NewLogSMSer()
		slog.Info("using log SMS sender (set TWILIO_ACCOUNT_SID to enable SMS)")
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, oauthRepo, tokenRepo, resetRepo, otpRepo, sellerProfileRepo, emailer, smser, cfg, blacklist, oauthStateStore)
	oauthService := service.NewOAuthService(userRepo, oauthRepo, tokenRepo, authService, cfg)

	// Initialize HTTP handlers
	httpHandler := httphandler.NewHandler(authService, oauthService, cfg)
	adminSvc := service.NewAdminService(userRepo)
	adminHandler := httphandler.NewAdminHandler(adminSvc, authService, cfg)
	prefsHandler := httphandler.NewPreferencesHandler(prefsRepo, authService)

	// Setup HTTP server
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/auth/admin/bootstrap", httpHandler.LoggingMiddleware(httpHandler.AdminBootstrap))
	mux.HandleFunc("/v1/auth/register", httphandler.IPRateLimit(rdb, "register")(httpHandler.LoggingMiddleware(httpHandler.Register)))
	mux.HandleFunc("/v1/auth/register/seller", httphandler.IPRateLimit(rdb, "register")(httpHandler.LoggingMiddleware(httpHandler.RegisterSeller)))
	mux.HandleFunc("/v1/auth/login", httphandler.IPRateLimit(rdb, "login")(httpHandler.LoggingMiddleware(httpHandler.Login)))
	mux.HandleFunc("/v1/auth/refresh", httpHandler.LoggingMiddleware(httpHandler.Refresh))
	mux.HandleFunc("/v1/auth/me", httpHandler.LoggingMiddleware(httpHandler.Me))
	mux.HandleFunc("/v1/auth/logout", httpHandler.LoggingMiddleware(httpHandler.Logout))
	mux.HandleFunc("/v1/auth/oauth/google/url", httpHandler.LoggingMiddleware(httpHandler.GoogleOAuthURL))
	mux.HandleFunc("/v1/auth/oauth/google/callback", httpHandler.LoggingMiddleware(httpHandler.GoogleOAuthCallback))
	mux.HandleFunc("/v1/auth/oauth/facebook/url", httpHandler.LoggingMiddleware(httpHandler.FacebookOAuthURL))
	mux.HandleFunc("/v1/auth/oauth/facebook/callback", httpHandler.LoggingMiddleware(httpHandler.FacebookOAuthCallback))
	mux.HandleFunc("/v1/auth/password/forgot", httpHandler.LoggingMiddleware(httpHandler.ForgotPassword))
	mux.HandleFunc("/v1/auth/password/reset", httphandler.IPRateLimit(rdb, "password-reset")(httpHandler.LoggingMiddleware(httpHandler.ResetPassword)))
	mux.HandleFunc("/v1/auth/password/forgot-otp", httphandler.IPRateLimit(rdb, "forgot-otp")(httpHandler.LoggingMiddleware(httpHandler.ForgotPasswordOTP)))
	mux.HandleFunc("/v1/auth/password/reset-otp", httphandler.IPRateLimit(rdb, "reset-otp")(httpHandler.LoggingMiddleware(httpHandler.ResetPasswordOTP)))
	mux.HandleFunc("/v1/auth/otp/send", httphandler.IPRateLimit(rdb, "otp-send")(httpHandler.LoggingMiddleware(httpHandler.SendOTP)))
	mux.HandleFunc("/v1/auth/otp/verify", httpHandler.LoggingMiddleware(httpHandler.VerifyOTP))

	// Swagger: spec served from the embedded swag doc (see docs/docs.go,
	// regenerated via `swag init -g cmd/main.go`), not a file on disk.
	// See pkg/swaggerx for why these two routes are split.
	mux.HandleFunc("/v1/docs/swagger.json", swaggerx.JSONHandler(""))
	mux.Handle("/v1/docs/", httpSwagger.Handler(httpSwagger.URL("/v1/docs/swagger.json")))

	// Health check endpoint
	// Admin routes — all require admin JWT
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /v1/admin/users", adminHandler.ListUsers)
	adminMux.HandleFunc("GET /v1/admin/users/{id}", adminHandler.GetUser)
	adminMux.HandleFunc("PUT /v1/admin/users/{id}/role", adminHandler.UpdateUserRole)
	adminMux.HandleFunc("DELETE /v1/admin/users/{id}", adminHandler.DeactivateUser)
	adminMux.HandleFunc("GET /v1/admin/sellers", adminHandler.ListSellers)
	adminMux.HandleFunc("PATCH /v1/admin/sellers/{id}/status", adminHandler.UpdateSellerStatus)
	mux.Handle("/v1/admin/", adminHandler.AdminAuthMiddleware(adminMux))

	// User preferences routes — auth validated inline by the handler
	mux.HandleFunc("GET /v1/users/me/preferences", httpHandler.LoggingMiddleware(prefsHandler.GetPreferences))
	mux.HandleFunc("PUT /v1/users/me/preferences", httpHandler.LoggingMiddleware(prefsHandler.SetPreferences))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})
	mux.Handle("/metrics", m.Handler())

	httpServer := &http.Server{
		Addr:            fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:         m.Middleware()(httpx.LimitBody(httpx.MaxBodyBytes)(mux)),
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
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
	grpcSrv := grpcx.NewServer()
	go func() {
		slog.Info("gRPC server configured", "port", cfg.GRPCPort)
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			slog.Error("Failed to listen on gRPC port", "error", err)
			sigChan <- syscall.SIGTERM
			return
		}
		grpcServer := grpcHandler.NewAuthServer(authService, cfg)
		authpb.RegisterAuthServiceServer(grpcSrv, grpcServer)
		reflection.Register(grpcSrv)
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcSrv.Serve(listener); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// ── Registry heartbeat ────────────────────────────────────────────────────
	svcCtx, svcCancel := context.WithCancel(context.Background())
	defer svcCancel()
	if rdb != nil {
		instanceID := uuid.New().String()
		addr := fmt.Sprintf("http://zapmarket-auth-service:%d", cfg.HTTPPort)
		go registry.Heartbeat(svcCtx, rdb, "auth-service", instanceID, addr, slog.Default())
		slog.Info("registered with gateway registry", "addr", addr)
	}

	// Wait for shutdown signal
	<-sigChan
	slog.Info("Shutdown signal received, gracefully shutting down")
	svcCancel()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	grpcSrv.GracefulStop()

	slog.Info("Auth service stopped")
}

func init() {
	// Configure structured logging; APP_ENV isn't available yet at init time
	// (config.Load runs in main), so default to development formatting here.
	slog.SetDefault(logger.New(os.Getenv("APP_ENV")))
}
