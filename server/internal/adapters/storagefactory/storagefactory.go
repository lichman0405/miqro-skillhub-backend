// Package storagefactory provides a unified storage factory that
// constructs the appropriate storage.Store from typed configuration.
package storagefactory

import (
	"context"
	"fmt"
	"log"

	"miqro-skillhub/server/internal/adapters/localstorage"
	"miqro-skillhub/server/internal/adapters/s3"
	"miqro-skillhub/server/internal/config"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// New constructs a storage.Store based on configuration.
// It logs the selected provider and returns a clear error when the
// provider or its config is invalid.
func New(ctx context.Context, cfg config.Config) (storage.Store, error) {
	switch cfg.StorageProvider {
	case "local":
		log.Printf("storage: using local filesystem storage (root=%s)", cfg.StorageRoot)
		store, err := localstorage.New(cfg.StorageRoot)
		if err != nil {
			return nil, fmt.Errorf("storagefactory: local: %w", err)
		}
		return store, nil

	case "s3":
		log.Printf("storage: using s3 storage (endpoint=%s, bucket=%s, region=%s, ssl=%v)",
			cfg.StorageEndpoint, cfg.StorageBucket, cfg.StorageRegion, cfg.StorageUseSSL)

		store, err := s3.New(ctx, s3.Config{
			Endpoint:  cfg.StorageEndpoint,
			Bucket:    cfg.StorageBucket,
			AccessKey: cfg.StorageAccessKey,
			SecretKey: cfg.StorageSecretKey,
			UseSSL:    cfg.StorageUseSSL,
			Region:    cfg.StorageRegion,
		})
		if err != nil {
			return nil, fmt.Errorf("storagefactory: s3: %w", err)
		}
		log.Printf("storage: s3 bucket %q verified", cfg.StorageBucket)
		return store, nil

	default:
		return nil, fmt.Errorf("storagefactory: unknown provider %q (must be \"local\" or \"s3\")", cfg.StorageProvider)
	}
}
