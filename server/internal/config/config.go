// Package config loads and validates the server configuration from
// environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// TrustedProxyCIDRs is a comma-separated list of CIDR blocks for reverse
	// proxies whose X-Forwarded-For header should be trusted.
	TrustedProxyCIDRs string
}

// TrustedProxyCIDRsList parses the comma-separated CIDR list.
func (c *Config) TrustedProxyCIDRsList() []string {
	if c.TrustedProxyCIDRs == "" {
		return nil
	}
	parts := strings.Split(c.TrustedProxyCIDRs, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Load reads configuration from the environment with sensible defaults.
func Load() (*Config, error) {
	localMode, err := parseBoolEnv("SKILLHUB_LOCAL_MODE", true)
	if err != nil {
		return nil, fmt.Errorf("config: SKILLHUB_LOCAL_MODE: %w", err)
	}

	cfg := &Config{
		APIAddr:            envOrDefault("SKILLHUB_API_ADDR", ":8080"),
		DatabaseURL:        envOrDefault("SKILLHUB_DATABASE_URL", "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"),
		RedisURL:           envOrDefault("SKILLHUB_REDIS_URL", "redis://localhost:6379/0"),
		StorageEndpoint:    envOrDefault("SKILLHUB_STORAGE_ENDPOINT", "localhost:9000"),
		StorageBucket:      envOrDefault("SKILLHUB_STORAGE_BUCKET", "skillhub"),
		StorageAccessKey:   envOrDefault("SKILLHUB_STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:   envOrDefault("SKILLHUB_STORAGE_SECRET_KEY", "minioadmin"),
		LocalMode:          localMode,
		CORSAllowedOrigins: os.Getenv("SKILLHUB_CORS_ALLOWED_ORIGINS"),
		TrustedProxyCIDRs:  os.Getenv("SKILLHUB_TRUSTED_PROXY_CIDRS"),
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

	// Production mode: reject known local-development defaults.
	return c.validateProduction()
}

// validateProduction rejects known weak defaults that must not reach production.
func (c *Config) validateProduction() error {
	defaultDB := "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
	if c.DatabaseURL == defaultDB {
		return fmt.Errorf("production mode: SKILLHUB_DATABASE_URL must not be the local development default")
	}
	if c.StorageAccessKey == "minioadmin" {
		return fmt.Errorf("production mode: SKILLHUB_STORAGE_ACCESS_KEY must not be the local development default (minioadmin)")
	}
	if c.StorageSecretKey == "minioadmin" {
		return fmt.Errorf("production mode: SKILLHUB_STORAGE_SECRET_KEY must not be the local development default (minioadmin)")
	}
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseBoolEnv(key string, defaultVal bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid boolean value %q for %s", v, key)
	}
	return b, nil
}
