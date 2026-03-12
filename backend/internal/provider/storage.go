package provider

import (
	"context"
	"io"
)

// StorageProvider defines the contract for file storage backends.
// Default implementation uses local filesystem.
// Plugins can replace with S3, MinIO, Alibaba OSS, etc.
type StorageProvider interface {
	// Save stores a file and returns its relative path.
	Save(ctx context.Context, filename string, reader io.Reader, size int64) (string, error)

	// Get retrieves a file by its relative path.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file by its relative path.
	Delete(ctx context.Context, path string) error

	// URL returns the public URL for a stored file.
	URL(path string) string

	// Exists checks whether a file exists.
	Exists(ctx context.Context, path string) (bool, error)
}
