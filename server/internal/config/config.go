// Package config loads and validates the server configuration from
// environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the server process.
type Config struct {
	// API address to listen on, e.g. ":8080".
	APIAddr string

	// Database connection URL (PostgreSQL).
	DatabaseURL string

	// Redis connection URL.
	RedisURL string

	// Object storage endpoint (S3-compatible, e.g. MinIO).
	StorageEndpoint string

	// Object storage bucket name.
	StorageBucket string

	// Object storage access key.
	StorageAccessKey string

	// Object storage secret key.
	StorageSecretKey string

	// LocalMode disables external dependency checks for local development.
	LocalMode bool

	// CORSAllowedOrigins is a comma-separated allowlist for browser clients.
	CORSAllowedOrigins string
}

// Load reads configuration from the environment with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		APIAddr:            envOrDefault("SKILLHUB_API_ADDR", ":8080"),
		DatabaseURL:        envOrDefault("SKILLHUB_DATABASE_URL", "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"),
		RedisURL:           envOrDefault("SKILLHUB_REDIS_URL", "redis://localhost:6379/0"),
		StorageEndpoint:    envOrDefault("SKILLHUB_STORAGE_ENDPOINT", "localhost:9000"),
		StorageBucket:      envOrDefault("SKILLHUB_STORAGE_BUCKET", "skillhub"),
		StorageAccessKey:   envOrDefault("SKILLHUB_STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:   envOrDefault("SKILLHUB_STORAGE_SECRET_KEY", "minioadmin"),
		LocalMode:          parseBoolEnv("SKILLHUB_LOCAL_MODE", true),
		CORSAllowedOrigins: os.Getenv("SKILLHUB_CORS_ALLOWED_ORIGINS"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required configuration values are present.
func (c *Config) validate() error {
	if c.LocalMode {
		return nil // skip validation in local mode
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("SKILLHUB_DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("SKILLHUB_REDIS_URL is required")
	}
	if c.StorageEndpoint == "" {
		return fmt.Errorf("SKILLHUB_STORAGE_ENDPOINT is required")
	}
	if c.StorageBucket == "" {
		return fmt.Errorf("SKILLHUB_STORAGE_BUCKET is required")
	}
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseBoolEnv(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
