// Package localstorage provides a local filesystem implementation of the
// storage.Store interface for development use.
package localstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"miqro-skillhub/server/sdk/skillhub/storage"
)

// Store implements storage.Store using the local filesystem.
type Store struct {
	root string
}

// New creates a local filesystem Store rooted at the given directory.
// The directory is created if it does not exist.
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("localstorage: create root: %w", err)
	}
	return &Store{root: root}, nil
}

// Root returns the filesystem root path.
func (s *Store) Root() string { return s.root }

// resolveKey resolves a storage key to a filesystem path and enforces
// that the resolved path stays within the store root (path traversal protection).
func (s *Store) resolveKey(key string) (string, error) {
	cleanRoot := filepath.Clean(s.root)
	fullPath := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(key)))
	if !strings.HasPrefix(fullPath, cleanRoot+string(filepath.Separator)) && fullPath != cleanRoot {
		return "", fmt.Errorf("localstorage: path traversal detected: %s", key)
	}
	return fullPath, nil
}

func (s *Store) PutObject(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	fullPath, err := s.resolveKey(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("localstorage: mkdir: %w", err)
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("localstorage: create: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("localstorage: write: %w", err)
	}
	return nil
}

func (s *Store) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	fullPath, err := s.resolveKey(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("localstorage: object not found: %s", key)
		}
		return nil, fmt.Errorf("localstorage: open: %w", err)
	}
	return f, nil
}

func (s *Store) DeleteObject(_ context.Context, key string) error {
	fullPath, err := s.resolveKey(key)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("localstorage: delete: %w", err)
	}
	return nil
}

func (s *Store) DeleteObjects(_ context.Context, keys []string) error {
	var errs []error
	for _, key := range keys {
		if err := s.DeleteObject(context.Background(), key); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("localstorage: delete objects: %v", errs)
	}
	return nil
}

func (s *Store) Exists(_ context.Context, key string) (bool, error) {
	fullPath, err := s.resolveKey(key)
	if err != nil {
		return false, err
	}
	_, statErr := os.Stat(fullPath)
	if statErr == nil {
		return true, nil
	}
	if os.IsNotExist(statErr) {
		return false, nil
	}
	return false, fmt.Errorf("localstorage: stat: %w", statErr)
}

func (s *Store) Metadata(_ context.Context, key string) (storage.ObjectMetadata, error) {
	fullPath, err := s.resolveKey(key)
	if err != nil {
		return storage.ObjectMetadata{}, err
	}
	fi, statErr := os.Stat(fullPath)
	if statErr != nil {
		return storage.ObjectMetadata{}, fmt.Errorf("localstorage: stat: %w", statErr)
	}
	return storage.ObjectMetadata{
		ContentType:   "application/octet-stream",
		ContentLength: fi.Size(),
		LastModified:  fi.ModTime(),
	}, nil
}

func (s *Store) PresignedURL(_ context.Context, key string, _ time.Duration, _ string) (string, error) {
	return "", errors.New("localstorage: presigned URLs not supported")
}

// Ensure Store implements the interface at compile time.
var _ storage.Store = (*Store)(nil)
