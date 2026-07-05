// Package s3 provides an S3-compatible implementation of the storage.Store
// interface for production use (AWS S3, MinIO, or compatible services).
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"miqro-skillhub/server/sdk/skillhub/storage"
)

// minioClient is the subset of the minio.Client API used by Store.
type minioClient interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
	BucketExists(ctx context.Context, bucketName string) (bool, error)
}

// Config holds the parameters needed to connect to an S3-compatible backend.
type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
}

// Store implements storage.Store against an S3-compatible backend.
type Store struct {
	client minioClient
	bucket string
}

// New creates an S3 Store connected to the given endpoint. It verifies
// that the target bucket exists and returns an error if it does not.
func New(ctx context.Context, cfg Config) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3: create client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("s3: bucket check failed: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("s3: bucket %q does not exist", cfg.Bucket)
	}

	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// Ping verifies connectivity by checking bucket existence.
func (s *Store) Ping(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("s3: ping: %w", err)
	}
	if !exists {
		return fmt.Errorf("s3: bucket %q does not exist", s.bucket)
	}
	return nil
}

func (s *Store) PutObject(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("s3: put object %q: %w", key, err)
	}
	return nil
}

func (s *Store) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3: get object %q: %w", key, err)
	}
	return obj, nil
}

func (s *Store) DeleteObject(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3: delete object %q: %w", key, err)
	}
	return nil
}

func (s *Store) DeleteObjects(ctx context.Context, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(keys))
	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	var errs []error
	for err := range s.client.RemoveObjects(ctx, s.bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if err.Err != nil {
			errs = append(errs, fmt.Errorf("s3: delete %q: %w", err.ObjectName, err.Err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("s3: delete objects: %v", errors.Join(errs...))
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && minioErr.StatusCode == 404 {
			return false, nil
		}
		// Also handle "The specified key does not exist" from S3.
		if minio.ToErrorResponse(err).StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("s3: stat object %q: %w", key, err)
	}
	return true, nil
}

func (s *Store) Metadata(ctx context.Context, key string) (storage.ObjectMetadata, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return storage.ObjectMetadata{}, fmt.Errorf("s3: stat object %q: %w", key, err)
	}
	return storage.ObjectMetadata{
		ContentType:   info.ContentType,
		ContentLength: info.Size,
		ETag:          info.ETag,
		LastModified:  info.LastModified,
	}, nil
}

func (s *Store) PresignedURL(ctx context.Context, key string, expiry time.Duration, downloadFilename string) (string, error) {
	reqParams := make(url.Values)
	if downloadFilename != "" {
		reqParams.Set("response-content-disposition",
			fmt.Sprintf(`attachment; filename="%s"`, downloadFilename))
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("s3: presigned url %q: %w", key, err)
	}
	return u.String(), nil
}

// Ensure Store implements the interface at compile time.
var _ storage.Store = (*Store)(nil)
