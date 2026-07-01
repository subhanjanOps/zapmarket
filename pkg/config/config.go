package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all service configuration
type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// JWT
	JWTSecretKey         string
	JWTRefreshSecretKey  string
	JWTAccessExpiryHours int
	JWTRefreshExpiryDays int

	// OAuth2 - Google
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// OAuth2 - Facebook
	FacebookClientID     string
	FacebookClientSecret string
	FacebookRedirectURL  string

	// Redis
	RedisURL string

	// Elasticsearch
	ElasticsearchURL string

	// MinIO / S3-compatible object storage. Endpoint is the internal address
	// (e.g. "minio:9000" inside Docker) used to talk to the server;
	// PublicURL is the externally-reachable address (e.g.
	// "http://localhost:9000") used to build URLs returned to API clients.
	// These differ under Docker Compose and must not be conflated.
	MinIOEndpoint  string
	MinIOPublicURL string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	// PaymentWebhookSecret signs/verifies the payment gateway's async
	// webhook callbacks (HMAC-SHA256 over the raw request body). Only
	// payment-service uses this.
	PaymentWebhookSecret string

	// Kafka
	KafkaBrokers []string

	// Downstream services
	AuthServiceAddr      string
	InventoryServiceAddr string
	PaymentServiceAddr   string
	CatalogServiceAddr   string

	// Service
	HTTPPort int
	GRPCPort int
	AppEnv   string

	// MigrateOnBoot runs pending migrations from ./migrations on startup
	// when true. Defaults to true for frictionless local dev; set
	// MIGRATE_ON_BOOT=false where migrations are run as an explicit deploy
	// step instead.
	MigrateOnBoot bool

	// AdminBootstrapSecret protects the one-shot admin creation endpoint.
	// Empty string disables the endpoint entirely.
	AdminBootstrapSecret string

	// PasswordResetBaseURL is the frontend URL prefix for reset links, e.g.
	// "https://app.zapmarket.com/reset-password". The token is appended as a
	// query parameter: ?token=<raw-token>.
	PasswordResetBaseURL string

	// SMTP settings for transactional email (password reset, verification).
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Resend (preferred transactional email). When set, ResendEmailer is used
	// instead of SMTPEmailer. RESEND_API_KEY="" falls back to LogEmailer.
	ResendAPIKey string
	EmailFrom    string // "From" address, e.g. "noreply@zapmarket.io"

	// Stripe payment gateway.
	StripeSecretKey      string // sk_test_... or sk_live_...
	StripeWebhookSecret  string // whsec_... for Stripe webhook signature verification

	// Twilio SMS for OTP delivery.
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioFromNumber  string // E.164, e.g. "+15005550006"

	// OTP settings.
	OTPExpiryMinutes int // default 10

	// OTLPEndpoint is the gRPC address of an OpenTelemetry collector
	// (e.g. "localhost:4317"). Empty string disables tracing.
	OTLPEndpoint string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Database defaults
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "zapuser"),
		DBPassword: getEnv("DB_PASSWORD", "zappass123"),
		DBName:     getEnv("DB_NAME", "userauth"),

		// JWT defaults
		JWTSecretKey:         getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
		JWTRefreshSecretKey:  getEnv("JWT_REFRESH_SECRET_KEY", "your-refresh-secret-key-change-in-production"),
		JWTAccessExpiryHours: getEnvInt("JWT_ACCESS_EXPIRY_HOURS", 1),
		JWTRefreshExpiryDays: getEnvInt("JWT_REFRESH_EXPIRY_DAYS", 7),

		// OAuth2
		GoogleClientID:     getEnv("OAUTH2_GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("OAUTH2_GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("OAUTH2_GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/oauth/google/callback"),

		FacebookClientID:     getEnv("OAUTH2_FACEBOOK_CLIENT_ID", ""),
		FacebookClientSecret: getEnv("OAUTH2_FACEBOOK_CLIENT_SECRET", ""),
		FacebookRedirectURL:  getEnv("OAUTH2_FACEBOOK_REDIRECT_URL", "http://localhost:8080/auth/oauth/facebook/callback"),

		// Redis
		RedisURL: getEnv("REDIS_URL", "localhost:6379"),

		// Elasticsearch
		ElasticsearchURL: getEnv("ES_URL", "http://localhost:9200"),

		// MinIO / S3-compatible object storage
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicURL: getEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "zapmarket"),
		MinIOUseSSL:    getEnvBool("MINIO_USE_SSL", false),

		PaymentWebhookSecret: getEnv("PAYMENT_WEBHOOK_SECRET", "your-webhook-secret-change-in-production"),

		// Kafka
		KafkaBrokers: getEnvStringSlice("KAFKA_BROKERS", []string{"localhost:29092"}),

		// Downstream services
		AuthServiceAddr:      getEnv("AUTH_SERVICE_ADDR", "localhost:50051"),
		InventoryServiceAddr: getEnv("INVENTORY_SERVICE_ADDR", "localhost:50053"),
		PaymentServiceAddr:   getEnv("PAYMENT_SERVICE_ADDR", "localhost:50054"),
		CatalogServiceAddr:   getEnv("CATALOG_SERVICE_ADDR", "localhost:50052"),

		// Service
		HTTPPort: getEnvInt("HTTP_PORT", 8080),
		GRPCPort: getEnvInt("GRPC_PORT", 50051),
		AppEnv:   getEnv("APP_ENV", "development"),

		MigrateOnBoot: getEnvBool("MIGRATE_ON_BOOT", true),

		AdminBootstrapSecret: getEnv("ADMIN_BOOTSTRAP_SECRET", ""),

		PasswordResetBaseURL: getEnv("PASSWORD_RESET_BASE_URL", "http://localhost:3000/reset-password"),

		SMTPHost:     getEnv("SMTP_HOST", "localhost"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@zapmarket.com"),

		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		EmailFrom:    getEnv("EMAIL_FROM", "onboarding@resend.dev"),

		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),

		TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),

		OTPExpiryMinutes: getEnvInt("OTP_EXPIRY_MINUTES", 10),

		OTLPEndpoint: getEnv("OTLP_ENDPOINT", ""),
	}

	// Validate required fields
	if cfg.DBUser == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.AppEnv == "production" {
		if cfg.JWTSecretKey == "your-secret-key-change-in-production" {
			return nil, fmt.Errorf("JWT_SECRET_KEY must be changed in production")
		}
		if cfg.JWTRefreshSecretKey == "your-refresh-secret-key-change-in-production" {
			return nil, fmt.Errorf("JWT_REFRESH_SECRET_KEY must be changed in production")
		}
		if cfg.DBPassword == "zappass123" {
			return nil, fmt.Errorf("DB_PASSWORD must be changed in production")
		}
		if cfg.MinIOAccessKey == "minioadmin" || cfg.MinIOSecretKey == "minioadmin" {
			return nil, fmt.Errorf("MINIO_ACCESS_KEY/MINIO_SECRET_KEY must be changed in production")
		}
		if cfg.PaymentWebhookSecret == "your-webhook-secret-change-in-production" {
			return nil, fmt.Errorf("PAYMENT_WEBHOOK_SECRET must be changed in production")
		}
	}

	return cfg, nil
}

// getEnv returns an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns an environment variable as int or a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool returns an environment variable as bool or a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getEnvStringSlice splits a comma-separated env var into a slice.
func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
