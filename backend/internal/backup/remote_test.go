package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockStorageProvider records uploads for testing.
type mockStorageProvider struct {
	mu       sync.Mutex
	uploads  map[string][]byte
	saveFail error
}

func newMockStorage() *mockStorageProvider {
	return &mockStorageProvider{uploads: make(map[string][]byte)}
}

func (m *mockStorageProvider) Save(_ context.Context, filename string, reader io.Reader, _ int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveFail != nil {
		return "", m.saveFail
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	m.uploads[filename] = data
	return filename, nil
}

func (m *mockStorageProvider) Get(_ context.Context, path string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.uploads[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorageProvider) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorageProvider) URL(path string) string {
	return "https://storage.example.com/" + path
}

func (m *mockStorageProvider) Exists(_ context.Context, path string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.uploads[path]
	return ok, nil
}

func setupTestService(t *testing.T) (*Service, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	svc := NewService(db, tmpDir, 10, tmpDir, "test")
	return svc, tmpDir
}

func TestUploadToRemote_Success(t *testing.T) {
	svc, tmpDir := setupTestService(t)

	// Create a test backup file
	testFile := "backup-test.db.gz"
	testData := []byte("fake backup data")
	err := os.WriteFile(filepath.Join(tmpDir, testFile), testData, 0644)
	require.NoError(t, err)

	storage := newMockStorage()

	remotePath, err := svc.UploadToRemote(context.Background(), testFile, storage, "backups/")
	require.NoError(t, err)
	assert.Equal(t, "backups/backup-test.db.gz", remotePath)

	// Verify the file was uploaded
	storage.mu.Lock()
	uploaded, ok := storage.uploads["backups/backup-test.db.gz"]
	storage.mu.Unlock()
	assert.True(t, ok)
	assert.Equal(t, testData, uploaded)
}

func TestUploadToRemote_FileNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	storage := newMockStorage()

	_, err := svc.UploadToRemote(context.Background(), "nonexistent.gz", storage, "backups/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open backup file")
}

func TestUploadToRemote_StorageError(t *testing.T) {
	svc, tmpDir := setupTestService(t)

	testFile := "backup-test.db.gz"
	err := os.WriteFile(filepath.Join(tmpDir, testFile), []byte("data"), 0644)
	require.NoError(t, err)

	storage := newMockStorage()
	storage.saveFail = fmt.Errorf("S3 connection timeout")

	_, err = svc.UploadToRemote(context.Background(), testFile, storage, "backups/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload to remote")
}
