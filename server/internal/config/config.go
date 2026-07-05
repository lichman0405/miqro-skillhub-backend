// Package config loads and validates the server configuration from
// environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the server process.
type Config struct {
	// API address to listen on, e.g. ":8080".
	APIAddr string

	// Database connection URL (PostgreSQL).
	DatabaseURL string

	// Redis connection URL. Used when SessionBackend=redis or
	// RateLimitBackend=redis.
	RedisURL string

	// StorageProvider selects the object storage backend: "local" or "s3".
	StorageProvider string

	// StorageRoot is the local filesystem storage root directory.
	StorageRoot string

	// Object storage endpoint (S3-compatible, e.g. MinIO).
	StorageEndpoint string

	// Object storage bucket name.
	StorageBucket string

	// Object storage access key.
	StorageAccessKey string

	// Object storage secret key.
	StorageSecretKey string

	// StorageUseSSL enables TLS for the S3 endpoint.
	StorageUseSSL bool

	// StorageRegion is the S3 region (default "us-east-1").
	StorageRegion string

	// AllowLocalStorageInProduction allows local storage when
	// SKILLHUB_LOCAL_MODE=false. This is an emergency override; prefer
	// S3 for production deployments.
	AllowLocalStorageInProduction bool

	// LocalMode disables external dependency checks for local development.
	LocalMode bool

	// CORSAllowedOrigins is a comma-separated allowlist for browser clients.
	CORSAllowedOrigins string

	// TrustedProxyCIDRs is a comma-separated list of CIDR blocks for reverse
	// proxies whose X-Forwarded-For header should be trusted.
	TrustedProxyCIDRs string

	// SessionBackend selects the session storage backend: "none" or "redis".
	SessionBackend string

	// SessionTTL is the time-to-live for server-side session records.
	SessionTTL time.Duration

	// SessionCookieSecure sets the Secure flag on the skillhub_session cookie.
	SessionCookieSecure bool

	// RateLimitBackend selects the rate-limit backend: "memory" or "redis".
	RateLimitBackend string
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

	storageUseSSL, err := parseBoolEnv("SKILLHUB_STORAGE_USE_SSL", false)
	if err != nil {
		return nil, fmt.Errorf("config: SKILLHUB_STORAGE_USE_SSL: %w", err)
	}

	allowLocalInProd, err := parseBoolEnv("SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION", false)
	if err != nil {
		return nil, fmt.Errorf("config: SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION: %w", err)
	}

	sessionCookieSecure, err := parseBoolEnv("SKILLHUB_SESSION_COOKIE_SECURE", false)
	if err != nil {
		return nil, fmt.Errorf("config: SKILLHUB_SESSION_COOKIE_SECURE: %w", err)
	}

	sessionTTL, err := parseDurationEnv("SKILLHUB_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("config: SKILLHUB_SESSION_TTL: %w", err)
	}

	cfg := &Config{
		APIAddr:    envOrDefault("SKILLHUB_API_ADDR", ":8080"),
		DatabaseURL: envOrDefault("SKILLHUB_DATABASE_URL", "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"),
		RedisURL:   envOrDefault("SKILLHUB_REDIS_URL", "redis://localhost:6379/0"),
		// Storage provider selection.
		StorageProvider:             envOrDefault("SKILLHUB_STORAGE_PROVIDER", "local"),
		StorageRoot:                 storageRoot(),
		StorageEndpoint:             envOrDefault("SKILLHUB_STORAGE_ENDPOINT", "localhost:9000"),
		StorageBucket:               envOrDefault("SKILLHUB_STORAGE_BUCKET", "skillhub"),
		StorageAccessKey:            envOrDefault("SKILLHUB_STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:            envOrDefault("SKILLHUB_STORAGE_SECRET_KEY", "minioadmin"),
		StorageUseSSL:                storageUseSSL,
		StorageRegion:                envOrDefault("SKILLHUB_STORAGE_REGION", "us-east-1"),
		AllowLocalStorageInProduction: allowLocalInProd,
		LocalMode:                     localMode,
		CORSAllowedOrigins:            os.Getenv("SKILLHUB_CORS_ALLOWED_ORIGINS"),
		TrustedProxyCIDRs:             os.Getenv("SKILLHUB_TRUSTED_PROXY_CIDRS"),
		SessionBackend:                envOrDefault("SKILLHUB_SESSION_BACKEND", "none"),
		SessionTTL:                    sessionTTL,
		SessionCookieSecure:           sessionCookieSecure,
		RateLimitBackend:              envOrDefault("SKILLHUB_RATE_LIMIT_BACKEND", "memory"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// storageRoot reads the canonical SKILLHUB_STORAGE_ROOT, falling back to
// the legacy STORAGE_ROOT variable, and finally to a default path.
func storageRoot() string {
	if v := os.Getenv("SKILLHUB_STORAGE_ROOT"); v != "" {
		return v
	}
	if v := os.Getenv("STORAGE_ROOT"); v != "" {
		return v
	}
	return "./data/storage"
}

// validate checks that required configuration values are present.
func (c *Config) validate() error {
	if c.LocalMode {
		return nil // skip validation in local mode
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("SKILLHUB_DATABASE_URL is required")
	}

	// Validate session and rate-limit backends.
	if err := c.validateSessionAndRateLimit(); err != nil {
		return err
	}

	// Validate storage provider selection and required fields.
	if err := c.validateStorage(); err != nil {
		return err
	}

	// Production mode: reject known local-development defaults.
	return c.validateProduction()
}

// validateSessionAndRateLimit validates session and rate-limit backend configuration.
func (c *Config) validateSessionAndRateLimit() error {
	switch c.SessionBackend {
	case "none", "redis":
	default:
		return fmt.Errorf("unknown session backend: %q (must be \"none\" or \"redis\")", c.SessionBackend)
	}
	switch c.RateLimitBackend {
	case "memory", "redis":
	default:
		return fmt.Errorf("unknown rate limit backend: %q (must be \"memory\" or \"redis\")", c.RateLimitBackend)
	}
	if c.SessionTTL <= 0 {
		return fmt.Errorf("SKILLHUB_SESSION_TTL must be positive")
	}
	if (c.SessionBackend == "redis" || c.RateLimitBackend == "redis") && c.RedisURL == "" {
		return fmt.Errorf("SKILLHUB_REDIS_URL is required when Redis-backed sessions or rate limiting are enabled")
	}
	return nil
}

// validateStorage validates the storage provider configuration.
func (c *Config) validateStorage() error {
	switch c.StorageProvider {
	case "local":
		if c.StorageRoot == "" {
			return fmt.Errorf("SKILLHUB_STORAGE_ROOT is required for local storage")
		}
		if !c.AllowLocalStorageInProduction {
			return fmt.Errorf("production mode: SKILLHUB_STORAGE_PROVIDER=local is not allowed unless SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION=true")
		}
		return nil
	case "s3":
		if c.StorageEndpoint == "" {
			return fmt.Errorf("SKILLHUB_STORAGE_ENDPOINT is required for s3 storage")
		}
		if c.StorageBucket == "" {
			return fmt.Errorf("SKILLHUB_STORAGE_BUCKET is required for s3 storage")
		}
		if c.StorageAccessKey == "" {
			return fmt.Errorf("SKILLHUB_STORAGE_ACCESS_KEY is required for s3 storage")
		}
		if c.StorageSecretKey == "" {
			return fmt.Errorf("SKILLHUB_STORAGE_SECRET_KEY is required for s3 storage")
		}
		return nil
	default:
		return fmt.Errorf("unknown storage provider: %q (must be \"local\" or \"s3\")", c.StorageProvider)
	}
}

// validateProduction rejects known weak defaults that must not reach production.
func (c *Config) validateProduction() error {
	defaultDB := "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
	if c.DatabaseURL == defaultDB {
		return fmt.Errorf("production mode: SKILLHUB_DATABASE_URL must not be the local development default")
	}
	// minioadmin credential check applies regardless of storage provider
	// (S3 access key may also be set while using local storage for other purposes).
	if c.StorageAccessKey == "minioadmin" {
		return fmt.Errorf("production mode: SKILLHUB_STORAGE_ACCESS_KEY must not be the local development default (minioadmin)")
	}
	if c.StorageSecretKey == "minioadmin" {
		return fmt.Errorf("production mode: SKILLHUB_STORAGE_SECRET_KEY must not be the local development default (minioadmin)")
	}
	// Redis-backed rate limiting is required for production multi-instance deployments.
	if c.RateLimitBackend != "redis" {
		return fmt.Errorf("production mode: SKILLHUB_RATE_LIMIT_BACKEND=redis is required")
	}
	// Redis-backed sessions are required for production multi-instance deployments.
	if c.SessionBackend != "redis" {
		return fmt.Errorf("production mode: SKILLHUB_SESSION_BACKEND=redis is required")
	}
	// Redis-backed sessions require secure cookies in production.
	if !c.SessionCookieSecure {
		return fmt.Errorf("production mode: SKILLHUB_SESSION_COOKIE_SECURE=true is required for Redis-backed sessions")
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

func parseDurationEnv(key string, defaultVal time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value %q for %s: %w", v, key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration value for %s must be positive", key)
	}
	return d, nil
}
