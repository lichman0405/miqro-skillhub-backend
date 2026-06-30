// Package storage defines the object storage abstraction used by the
// SDK.  It mirrors the Java source ObjectStorageService interface
// exactly so that skill file uploads, downloads, bundle management,
// and presigned URLs work identically regardless of the backing
// provider (local filesystem for development, S3/MinIO for
// production).
//
// Source reference:
//
//	com.iflytek.skillhub.storage.ObjectStorageService
//	server/skillhub-storage/src/main/java/com/iflytek/skillhub/storage/ObjectStorageService.java
package storage

import (
	"context"
	"io"
	"time"
)

// ObjectMetadata carries key-level metadata returned by the storage
// backend.
type ObjectMetadata struct {
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  time.Time
}

// Store is the storage abstraction contract.  Every implementation
// (local filesystem, S3, MinIO) must satisfy this interface.
type Store interface {
	// PutObject stores data at key with the given size and content type.
	PutObject(ctx context.Context, key string, data io.Reader, size int64, contentType string) error

	// GetObject retrieves the data stored at key.
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)

	// DeleteObject removes the object at key.
	DeleteObject(ctx context.Context, key string) error

	// DeleteObjects removes multiple objects in a single call.
	DeleteObjects(ctx context.Context, keys []string) error

	// Exists reports whether an object exists at key.
	Exists(ctx context.Context, key string) (bool, error)

	// Metadata returns metadata for the object at key.
	Metadata(ctx context.Context, key string) (ObjectMetadata, error)

	// PresignedURL generates a time-limited download URL for the
	// object at key.
	PresignedURL(ctx context.Context, key string, expiry time.Duration, downloadFilename string) (string, error)
}
