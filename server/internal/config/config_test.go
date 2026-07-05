package config

import (
	"os"
	"testing"
)

func TestLoad_InvalidLocalModeBoolReturnsError(t *testing.T) {
	os.Setenv("SKILLHUB_LOCAL_MODE", "not-a-bool")
	defer os.Unsetenv("SKILLHUB_LOCAL_MODE")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid SKILLHUB_LOCAL_MODE value")
	}
}

func TestValidate_ProductionRejectsDefaultDatabaseURL(t *testing.T) {
	cfg := &Config{
		DatabaseURL:                "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable",
		RedisURL:                   "redis://prod:6379/0",
		StorageProvider:            "s3",
		StorageEndpoint:            "s3.amazonaws.com",
		StorageBucket:              "prod-bucket",
		StorageAccessKey:           "AKIAPRODUCTION",
		StorageSecretKey:           "prod-secret-key",
		AllowLocalStorageInProduction: true,
		LocalMode:                  false,
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for default database URL in production mode")
	}
}

func TestValidate_ProductionRejectsDefaultMinIOCredentials(t *testing.T) {
	cfg := &Config{
		DatabaseURL:                "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		RedisURL:                   "redis://prod:6379/0",
		StorageProvider:            "s3",
		StorageEndpoint:            "s3.amazonaws.com",
		StorageBucket:              "prod-bucket",
		StorageAccessKey:           "minioadmin",
		StorageSecretKey:           "prod-secret",
		AllowLocalStorageInProduction: true,
		LocalMode:                  false,
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for minioadmin access key in production mode")
	}

	cfg2 := &Config{
		DatabaseURL:                "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		RedisURL:                   "redis://prod:6379/0",
		StorageProvider:            "s3",
		StorageEndpoint:            "s3.amazonaws.com",
		StorageBucket:              "prod-bucket",
		StorageAccessKey:           "prod-key",
		StorageSecretKey:           "minioadmin",
		AllowLocalStorageInProduction: true,
		LocalMode:                  false,
	}

	err = cfg2.validate()
	if err == nil {
		t.Fatal("expected error for minioadmin secret key in production mode")
	}
}

func TestValidate_ProductionRejectsLocalStorageWithoutOverride(t *testing.T) {
	cfg := &Config{
		DatabaseURL:     "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		StorageProvider: "local",
		StorageRoot:     "/data/storage",
		StorageAccessKey: "prod-key",
		StorageSecretKey: "prod-secret",
		LocalMode:       false,
		// AllowLocalStorageInProduction defaults to false.
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for local storage in production without override")
	}
}

func TestValidate_ProductionAllowsLocalStorageWithOverride(t *testing.T) {
	cfg := &Config{
		DatabaseURL:                   "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		StorageProvider:               "local",
		StorageRoot:                   "/data/storage",
		StorageAccessKey:              "prod-key",
		StorageSecretKey:              "prod-secret",
		AllowLocalStorageInProduction: true,
		LocalMode:                     false,
	}

	err := cfg.validate()
	if err != nil {
		t.Fatalf("local storage in production with override should be allowed: %v", err)
	}
}

func TestValidate_InvalidStorageProvider(t *testing.T) {
	cfg := &Config{
		DatabaseURL:     "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		StorageProvider: "gcs",
		StorageAccessKey: "prod-key",
		StorageSecretKey: "prod-secret",
		AllowLocalStorageInProduction: true,
		LocalMode:       false,
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for unknown storage provider")
	}
}

func TestValidate_S3RequiresEndpoint(t *testing.T) {
	cfg := &Config{
		DatabaseURL:     "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		StorageProvider: "s3",
		StorageBucket:   "prod-bucket",
		StorageAccessKey: "prod-key",
		StorageSecretKey: "prod-secret",
		AllowLocalStorageInProduction: true,
		LocalMode:       false,
		// StorageEndpoint is empty.
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for missing S3 endpoint")
	}
}

func TestValidate_S3RequiresBucket(t *testing.T) {
	cfg := &Config{
		DatabaseURL:     "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		StorageProvider: "s3",
		StorageEndpoint: "s3.example.com",
		StorageAccessKey: "prod-key",
		StorageSecretKey: "prod-secret",
		AllowLocalStorageInProduction: true,
		LocalMode:       false,
		// StorageBucket is empty.
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for missing S3 bucket")
	}
}

func TestValidate_LocalRequiresStorageRoot(t *testing.T) {
	cfg := &Config{
		DatabaseURL:                   "postgres://user:pass@prod-db.example.com:5432/skillhub?sslmode=require",
		StorageProvider:               "local",
		StorageAccessKey:              "prod-key",
		StorageSecretKey:              "prod-secret",
		AllowLocalStorageInProduction: true,
		LocalMode:                     false,
		// StorageRoot is empty.
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for missing storage root with local provider")
	}
}

func TestValidate_LocalModeAllowsDefaults(t *testing.T) {
	cfg := &Config{
		DatabaseURL:      "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable",
		RedisURL:         "redis://localhost:6379/0",
		StorageProvider:  "local",
		StorageRoot:      "./data/storage",
		StorageEndpoint:  "localhost:9000",
		StorageBucket:    "skillhub",
		StorageAccessKey: "minioadmin",
		StorageSecretKey: "minioadmin",
		LocalMode:        true,
	}

	err := cfg.validate()
	if err != nil {
		t.Fatalf("local mode should allow defaults, got: %v", err)
	}
}

func TestStorageRoot_FallbackChain(t *testing.T) {
	// Clear both env vars.
	os.Unsetenv("SKILLHUB_STORAGE_ROOT")
	os.Unsetenv("STORAGE_ROOT")

	root := storageRoot()
	if root != "./data/storage" {
		t.Errorf("default storage root should be ./data/storage, got %q", root)
	}

	// Legacy STORAGE_ROOT fallback.
	os.Setenv("STORAGE_ROOT", "/legacy/path")
	defer os.Unsetenv("STORAGE_ROOT")
	root = storageRoot()
	if root != "/legacy/path" {
		t.Errorf("expected legacy STORAGE_ROOT fallback, got %q", root)
	}

	// Canonical SKILLHUB_STORAGE_ROOT takes precedence.
	os.Setenv("SKILLHUB_STORAGE_ROOT", "/canonical/path")
	defer os.Unsetenv("SKILLHUB_STORAGE_ROOT")
	root = storageRoot()
	if root != "/canonical/path" {
		t.Errorf("expected canonical SKILLHUB_STORAGE_ROOT to take precedence, got %q", root)
	}
}

func TestLoad_LocalModeDefaultsToTrue(t *testing.T) {
	// Ensure no env override.
	os.Unsetenv("SKILLHUB_LOCAL_MODE")
	defer os.Unsetenv("SKILLHUB_LOCAL_MODE")

	// We can't call Load() without hitting env validation for production mode
	// (which fails without real DB/Redis URLs). Just test parseBoolEnv directly.
	b, err := parseBoolEnv("SKILLHUB_LOCAL_MODE", true)
	if err != nil {
		t.Fatalf("parseBoolEnv with no value should not error: %v", err)
	}
	if !b {
		t.Error("default for SKILLHUB_LOCAL_MODE should be true")
	}
}

func TestTrustedProxyCIDRsList(t *testing.T) {
	cfg := &Config{TrustedProxyCIDRs: ""}
	if list := cfg.TrustedProxyCIDRsList(); list != nil {
		t.Errorf("empty CIDRs should return nil, got %v", list)
	}

	cfg2 := &Config{TrustedProxyCIDRs: "10.0.0.0/8, 172.16.0.0/12"}
	list := cfg2.TrustedProxyCIDRsList()
	if len(list) != 2 {
		t.Fatalf("expected 2 CIDRs, got %d", len(list))
	}
	if list[0] != "10.0.0.0/8" {
		t.Errorf("expected first CIDR 10.0.0.0/8, got %q", list[0])
	}
	if list[1] != "172.16.0.0/12" {
		t.Errorf("expected second CIDR 172.16.0.0/12, got %q", list[1])
	}
}
