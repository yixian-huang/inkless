package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"blotting-consultancy/internal/provider"
)

// RemoteBackupConfig holds configuration for remote backup storage.
type RemoteBackupConfig struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"` // "s3", "oss", etc.
	Prefix   string `json:"prefix"`   // e.g. "backups/site-1/"
}

// UploadToRemote uploads a backup file to the configured remote storage provider.
// Returns the remote path where the file was stored.
func (s *Service) UploadToRemote(ctx context.Context, filename string, remote provider.StorageProvider, prefix string) (string, error) {
	localPath := filepath.Join(s.backupDir, filename)
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open backup file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat backup file: %w", err)
	}

	remotePath := prefix + filename
	_, err = remote.Save(ctx, remotePath, f, info.Size())
	if err != nil {
		return "", fmt.Errorf("upload to remote: %w", err)
	}

	return remotePath, nil
}
