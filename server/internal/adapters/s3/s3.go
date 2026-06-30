// Package s3 provides an S3-compatible implementation of the storage.Store
// interface for production use (AWS S3, MinIO, or compatible services).
//
// Phase 05 provides the adapter skeleton. The full implementation (AWS SDK,
// multipart uploads, presigned URLs) is completed when deployment
// configuration is available in Phase 08.
package s3

import (
	"context"
	"errors"
	"io"
	"time"

	"miqro-skillhub/server/sdk/skillhub/storage"
)

// Store implements storage.Store against an S3-compatible backend.
type Store struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
	connected bool
}

// New creates an S3 Store. Returns an unconnected store — call Ping to verify.
func New(endpoint, bucket, accessKey, secretKey string) *Store {
	return &Store{
		endpoint:  endpoint,
		bucket:    bucket,
		accessKey: accessKey,
		secretKey: secretKey,
	}
}

// Ping verifies connectivity. Phase 05 returns "not connected" until the
// AWS SDK is wired in Phase 08.
func (s *Store) Ping() string {
	return "not connected — S3 adapter ready for Phase 08 wiring"
}

// IsAvailable reports whether the store is connected.
func (s *Store) IsAvailable() bool {
	return s.connected
}

func (s *Store) PutObject(_ context.Context, key string, data io.Reader, size int64, contentType string) error {
	return errors.New("s3: not connected — full implementation in Phase 08")
}

func (s *Store) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	return nil, errors.New("s3: not connected")
}

func (s *Store) DeleteObject(_ context.Context, key string) error {
	return errors.New("s3: not connected")
}

func (s *Store) DeleteObjects(_ context.Context, keys []string) error {
	return errors.New("s3: not connected")
}

func (s *Store) Exists(_ context.Context, key string) (bool, error) {
	return false, errors.New("s3: not connected")
}

func (s *Store) Metadata(_ context.Context, key string) (storage.ObjectMetadata, error) {
	return storage.ObjectMetadata{}, errors.New("s3: not connected")
}

func (s *Store) PresignedURL(_ context.Context, key string, expiry time.Duration, downloadFilename string) (string, error) {
	return "", errors.New("s3: not connected")
}

// Ensure Store implements the interface at compile time.
var _ storage.Store = (*Store)(nil)
