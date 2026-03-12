package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements provider.StorageProvider using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local filesystem storage provider.
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

// Save stores a file to the local filesystem.
func (s *LocalStorage) Save(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	fullPath := filepath.Join(s.baseDir, filename)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filename, nil
}

// Get retrieves a file from the local filesystem.
func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, path)
	return os.Open(fullPath)
}

// Delete removes a file from the local filesystem.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.baseDir, path)
	return os.Remove(fullPath)
}

// URL returns the public URL prefix for a stored file.
func (s *LocalStorage) URL(path string) string {
	return filepath.Join(s.baseDir, path)
}

// Exists checks whether a file exists on the local filesystem.
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.baseDir, path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
