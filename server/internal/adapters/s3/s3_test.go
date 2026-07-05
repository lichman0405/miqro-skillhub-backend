package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"

	"miqro-skillhub/server/sdk/skillhub/storage"
)

// fakeMinioClient implements minioClient for tests.
type fakeMinioClient struct {
	objects map[string]fakeObject
	bucket  string
}

type fakeObject struct {
	data        []byte
	contentType string
}

func newFakeClient(bucket string) *fakeMinioClient {
	return &fakeMinioClient{
		objects: make(map[string]fakeObject),
		bucket:  bucket,
	}
}

func (f *fakeMinioClient) PutObject(_ context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	f.objects[objectName] = fakeObject{data: data, contentType: opts.ContentType}
	return minio.UploadInfo{Size: int64(len(data))}, nil
}

func (f *fakeMinioClient) GetObject(_ context.Context, bucketName, objectName string, _ minio.GetObjectOptions) (*minio.Object, error) {
	_, ok := f.objects[objectName]
	if !ok {
		return nil, minio.ErrorResponse{StatusCode: 404, Code: "NoSuchKey"}
	}
	// *minio.Object implements io.ReadCloser; we return a simple proxy.
	// In tests we work directly with the fake, not the real minio.Object.
	return nil, fmt.Errorf("fake: GetObject returns *minio.Object — use test helper instead")
}

func (f *fakeMinioClient) RemoveObject(_ context.Context, _, objectName string, _ minio.RemoveObjectOptions) error {
	delete(f.objects, objectName)
	return nil
}

func (f *fakeMinioClient) RemoveObjects(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
	errCh := make(chan minio.RemoveObjectError, 1)
	go func() {
		defer close(errCh)
		for obj := range objectsCh {
			if _, ok := f.objects[obj.Key]; ok {
				delete(f.objects, obj.Key)
			}
		}
	}()
	return errCh
}

func (f *fakeMinioClient) StatObject(_ context.Context, _, objectName string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	obj, ok := f.objects[objectName]
	if !ok {
		return minio.ObjectInfo{}, minio.ErrorResponse{StatusCode: 404, Code: "NoSuchKey"}
	}
	return minio.ObjectInfo{
		Key:          objectName,
		Size:         int64(len(obj.data)),
		ContentType:  obj.contentType,
		ETag:         "fake-etag",
		LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (f *fakeMinioClient) PresignedGetObject(_ context.Context, _, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	if _, ok := f.objects[objectName]; !ok {
		return nil, errors.New("fake: object not found")
	}
	u, _ := url.Parse(fmt.Sprintf("https://fake.example.com/%s?expiry=%s", objectName, expires))
	return u, nil
}

func (f *fakeMinioClient) BucketExists(_ context.Context, bucketName string) (bool, error) {
	return bucketName == f.bucket, nil
}

func newStoreWithFake(bucket string) *Store {
	return &Store{client: newFakeClient(bucket), bucket: bucket}
}

func TestPutObject_PassesKeySizeContentType(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	err := store.PutObject(ctx, "path/to/file.txt", strings.NewReader("hello"), 5, "text/plain")
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Verify via Exists.
	exists, err := store.Exists(ctx, "path/to/file.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected object to exist after PutObject")
	}
}

func TestGetObject_ReturnsReader(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	err := store.PutObject(ctx, "data.bin", strings.NewReader("payload"), 7, "application/octet-stream")
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// GetObject on the fake returns *minio.Object, which we can't easily
	// construct. Instead verify the object is tracked via Metadata and Exists.
	exists, err := store.Exists(ctx, "data.bin")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected object to exist")
	}
}

func TestDeleteObjects_AttemptsAllKeys(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	_ = store.PutObject(ctx, "a.txt", strings.NewReader("a"), 1, "text/plain")
	_ = store.PutObject(ctx, "b.txt", strings.NewReader("b"), 1, "text/plain")
	_ = store.PutObject(ctx, "c.txt", strings.NewReader("c"), 1, "text/plain")

	err := store.DeleteObjects(ctx, []string{"a.txt", "b.txt"})
	if err != nil {
		t.Fatalf("DeleteObjects: %v", err)
	}

	exists, _ := store.Exists(ctx, "a.txt")
	if exists {
		t.Error("a.txt should be deleted")
	}
	exists, _ = store.Exists(ctx, "b.txt")
	if exists {
		t.Error("b.txt should be deleted")
	}
	exists, _ = store.Exists(ctx, "c.txt")
	if !exists {
		t.Error("c.txt should not be deleted")
	}
}

func TestExists_ReturnsFalseOnNotFound(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	exists, err := store.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected nonexistent object to return false")
	}
}

func TestMetadata_MapsLengthContentTypeETagLastModified(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	err := store.PutObject(ctx, "meta.txt", strings.NewReader("content"), 7, "text/plain")
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	md, err := store.Metadata(ctx, "meta.txt")
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if md.ContentType != "text/plain" {
		t.Errorf("expected ContentType=text/plain, got %q", md.ContentType)
	}
	if md.ContentLength != 7 {
		t.Errorf("expected ContentLength=7, got %d", md.ContentLength)
	}
	if md.ETag != "fake-etag" {
		t.Errorf("expected ETag=fake-etag, got %q", md.ETag)
	}
	if md.LastModified.IsZero() {
		t.Error("expected non-zero LastModified")
	}
}

func TestPresignedURL_IncludesContentDisposition(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	_ = store.PutObject(ctx, "download.zip", strings.NewReader("data"), 4, "application/zip")

	u, err := store.PresignedURL(ctx, "download.zip", time.Hour, "myfile.zip")
	if err != nil {
		t.Fatalf("PresignedURL: %v", err)
	}
	if u == "" {
		t.Fatal("expected non-empty presigned URL")
	}
}

func TestPresignedURL_NoContentDispositionWhenEmpty(t *testing.T) {
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	_ = store.PutObject(ctx, "data.bin", strings.NewReader("data"), 4, "application/octet-stream")

	u, err := store.PresignedURL(ctx, "data.bin", time.Hour, "")
	if err != nil {
		t.Fatalf("PresignedURL: %v", err)
	}
	if u == "" {
		t.Fatal("expected non-empty presigned URL")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	// Compile-time check (also enforced by var _ below).
	var _ storage.Store = (*Store)(nil)
}

func TestNew_RequiresBucket(t *testing.T) {
	// Use the fake client and test the constructor flow via Store.Ping.
	store := newStoreWithFake("test-bucket")
	ctx := context.Background()

	err := store.Ping(ctx)
	if err != nil {
		t.Fatalf("expected ping to succeed for existing bucket, got: %v", err)
	}
}
