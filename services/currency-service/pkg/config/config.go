package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all currency-service configuration.
type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// Service
	HTTPPort int
	GRPCPort int
	AppEnv   string

	// Redis (shared instance with gateway)
	RedisURL string

	// FX providers
	RateProviderURL         string
	FallbackRateProviderURL string
	RateRefreshInterval     time.Duration
	MaxRateAge              time.Duration

	// Registry
	ServiceName string

	// MigrateOnBoot runs migrations at startup (default: true).
	MigrateOnBoot bool
}

func Load() (*Config, error) {
	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "zapuser"),
		DBPassword: getEnv("DB_PASSWORD", "zappass123"),
		DBName:     getEnv("DB_NAME", "currency"),

		HTTPPort: getEnvInt("HTTP_PORT", 8086),
		GRPCPort: getEnvInt("GRPC_PORT", 50056),
		AppEnv:   getEnv("APP_ENV", "development"),

		RedisURL: getEnv("REDIS_URL", "localhost:6379"),

		RateProviderURL:         getEnv("RATE_PROVIDER_URL", "https://api.frankfurter.app"),
		FallbackRateProviderURL: getEnv("FALLBACK_RATE_PROVIDER_URL", "https://open.er-api.com"),
		RateRefreshInterval:     getEnvDuration("RATE_REFRESH_INTERVAL", time.Hour),
		MaxRateAge:              getEnvDuration("MAX_RATE_AGE", 24*time.Hour),

		ServiceName:   "currency-service",
		MigrateOnBoot: getEnvBool("MIGRATE_ON_BOOT", true),
	}

	if cfg.DBUser == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
