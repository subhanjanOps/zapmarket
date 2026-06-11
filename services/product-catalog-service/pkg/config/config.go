package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// JWT
	JWTSecretKey string

	// Redis
	RedisURL string

	// Elasticsearch
	ElasticsearchURL string

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
		JWTSecretKey: getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),

		// Redis URL
		RedisURL: getEnv("REDIS_URL", "localhost:6379"),

		// Elasticsearch URL
		ElasticsearchURL: getEnv("ES_URL", "http://localhost:9200"),

		// Service
		HTTPPort: getEnvInt("HTTP_PORT", 8081),
		GRPCPort: getEnvInt("GRPC_PORT", 50052),
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
