package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ── fakes ──────────────────────────────────────────────────────────────────

type fakeFileRepo struct {
	files []skill.SkillFile
	err   error
}

func (f *fakeFileRepo) FindByVersionID(_ context.Context, _ int64) ([]skill.SkillFile, error) {
	return f.files, f.err
}

func (f *fakeFileRepo) Save(_ context.Context, _ skill.SkillFile) (skill.SkillFile, error) {
	panic("not implemented")
}
func (f *fakeFileRepo) SaveAll(_ context.Context, _ []skill.SkillFile) ([]skill.SkillFile, error) {
	panic("not implemented")
}
func (f *fakeFileRepo) DeleteByVersionID(_ context.Context, _ int64) error {
	panic("not implemented")
}

type fakeObjStore struct {
	data   map[string][]byte
	getErr error
	reader io.ReadCloser // custom reader; if set, GetObject returns this
}

func newFakeObjStore() *fakeObjStore {
	return &fakeObjStore{data: make(map[string][]byte)}
}

func (s *fakeObjStore) PutObject(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	b, _ := io.ReadAll(data)
	s.data[key] = b
	return nil
}
func (s *fakeObjStore) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.reader != nil {
		return s.reader, nil
	}
	b, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (s *fakeObjStore) DeleteObject(_ context.Context, _ string) error              { return nil }
func (s *fakeObjStore) DeleteObjects(_ context.Context, _ []string) error            { return nil }
func (s *fakeObjStore) Exists(_ context.Context, key string) (bool, error)           { _, ok := s.data[key]; return ok, nil }
func (s *fakeObjStore) Metadata(_ context.Context, _ string) (storage.ObjectMetadata, error) {
	return storage.ObjectMetadata{}, nil
}
func (s *fakeObjStore) PresignedURL(_ context.Context, _ string, _ time.Duration, _ string) (string, error) {
	return "", errors.New("not supported")
}

// ── errorReader ────────────────────────────────────────────────────────────

type errorReader struct{ err error }

func (e *errorReader) Read([]byte) (int, error) { return 0, e.err }

type closeErrorReader struct {
	*bytes.Reader
	closeErr error
}

func (c *closeErrorReader) Close() error { return c.closeErr }

// ── tests ──────────────────────────────────────────────────────────────────

func TestReadPackageFileEntries_NormalRead(t *testing.T) {
	store := newFakeObjStore()
	_ = store.PutObject(context.Background(), "key1", bytes.NewReader([]byte("hello")), 5, "text/plain")

	repo := &fakeFileRepo{
		files: []skill.SkillFile{
			{FilePath: "a.txt", StorageKey: "key1", FileSize: 5, ContentType: "text/plain"},
			{FilePath: "b.txt", StorageKey: "", FileSize: 0, ContentType: "text/plain"},
		},
	}

	entries, err := readPackageFileEntries(context.Background(), repo, store, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if string(entries[0].Content) != "hello" {
		t.Errorf("expected content 'hello', got %q", string(entries[0].Content))
	}
	if entries[1].Content != nil {
		t.Errorf("expected nil content for empty StorageKey, got %v", entries[1].Content)
	}

	// Verify returned type satisfies agentci.PackageFileEntry.
	var _ []agentci.PackageFileEntry = entries
}

func TestReadPackageFileEntries_GetObjectError(t *testing.T) {
	store := newFakeObjStore()
	store.getErr = errors.New("s3: connection refused")

	repo := &fakeFileRepo{
		files: []skill.SkillFile{
			{FilePath: "a.txt", StorageKey: "key1", FileSize: 5},
		},
	}

	_, err := readPackageFileEntries(context.Background(), repo, store, 1)
	if err == nil {
		t.Fatal("expected error from GetObject, got nil")
	}
}

func TestReadPackageFileEntries_ReadAllError(t *testing.T) {
	store := newFakeObjStore()
	readErr := errors.New("read failed")
	store.reader = io.NopCloser(&errorReader{err: readErr})

	repo := &fakeFileRepo{
		files: []skill.SkillFile{
			{FilePath: "a.txt", StorageKey: "key1", FileSize: 5},
		},
	}

	_, err := readPackageFileEntries(context.Background(), repo, store, 1)
	if err == nil {
		t.Fatal("expected error from io.ReadAll, got nil")
	}
}

func TestReadPackageFileEntries_CloseError(t *testing.T) {
	store := newFakeObjStore()
	closeErr := errors.New("close failed")
	store.reader = &closeErrorReader{
		Reader:   bytes.NewReader([]byte("hello")),
		closeErr: closeErr,
	}

	repo := &fakeFileRepo{
		files: []skill.SkillFile{
			{FilePath: "a.txt", StorageKey: "key1", FileSize: 5},
		},
	}

	_, err := readPackageFileEntries(context.Background(), repo, store, 1)
	if err == nil {
		t.Fatal("expected error from Close, got nil")
	}
}

func TestReadPackageFileEntries_NoStorageKeyReturnsNilContent(t *testing.T) {
	store := newFakeObjStore()
	repo := &fakeFileRepo{
		files: []skill.SkillFile{
			{FilePath: "manifest.yaml", StorageKey: "", FileSize: 0},
		},
	}

	entries, err := readPackageFileEntries(context.Background(), repo, store, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != nil {
		t.Errorf("expected nil content, got %v", entries[0].Content)
	}
}
