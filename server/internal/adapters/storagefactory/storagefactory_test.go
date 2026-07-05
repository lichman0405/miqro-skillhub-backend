package storagefactory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"miqro-skillhub/server/internal/config"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

func TestNew_LocalProvider(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "storage")

	cfg := config.Config{
		StorageProvider: "local",
		StorageRoot:     root,
	}

	store, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	// Store should be a *localstorage.Store and root should exist.
	localStore, ok := store.(interface{ Root() string })
	if !ok {
		t.Fatal("expected store to have Root() method (local storage)")
	}
	if localStore.Root() != root {
		t.Errorf("expected root=%s, got %s", root, localStore.Root())
	}

	// Verify root directory was created.
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Errorf("expected root directory to exist: err=%v", err)
	}
}

func TestNew_S3ProviderWithValidConfig_CallsConstructor(t *testing.T) {
	// S3 constructor requires a live endpoint; test that a connection
	// failure produces a clear error from the storage factory, not a
	// silent fallback.
	cfg := config.Config{
		StorageProvider:  "s3",
		StorageEndpoint:  "localhost:19999", // no server here
		StorageBucket:    "test-bucket",
		StorageAccessKey: "test-key",
		StorageSecretKey: "test-secret",
		StorageUseSSL:    false,
		StorageRegion:    "us-east-1",
	}

	_, err := New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error connecting to nonexistent S3 endpoint")
	}
}

func TestNew_InvalidProvider(t *testing.T) {
	cfg := config.Config{
		StorageProvider: "invalid",
	}

	_, err := New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestNew_InterfaceReturned(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		StorageProvider: "local",
		StorageRoot:     filepath.Join(dir, "storage"),
	}

	store, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var _ storage.Store = store
}
