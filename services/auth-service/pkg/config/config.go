package config

import (
	"fmt"
	"os"
	"strconv"
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

	// Service
	HTTPPort int
	GRPCPort int
	AppEnv   string
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

		// Service
		HTTPPort: getEnvInt("HTTP_PORT", 8080),
		GRPCPort: getEnvInt("GRPC_PORT", 50051),
		AppEnv:   getEnv("APP_ENV", "development"),
	}

	// Validate required fields
	if cfg.DBUser == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.JWTSecretKey == "your-secret-key-change-in-production" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("JWT_SECRET_KEY must be changed in production")
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
