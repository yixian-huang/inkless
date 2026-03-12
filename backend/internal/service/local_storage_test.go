package service_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/service"
)

// Verify LocalStorage implements StorageProvider at compile time.
var _ provider.StorageProvider = (*service.LocalStorage)(nil)

func TestLocalStorageSaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	storage := service.NewLocalStorage(tmpDir)

	ctx := context.Background()
	content := []byte("hello world")
	path, err := storage.Save(ctx, "test.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists on disk
	if _, err := os.Stat(filepath.Join(tmpDir, path)); err != nil {
		t.Fatalf("file not found on disk: %v", err)
	}

	// Get the file back
	reader, err := storage.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(got))
	}
}

func TestLocalStorageDelete(t *testing.T) {
	tmpDir := t.TempDir()
	storage := service.NewLocalStorage(tmpDir)

	ctx := context.Background()
	content := []byte("delete me")
	path, _ := storage.Save(ctx, "del.txt", bytes.NewReader(content), int64(len(content)))

	if err := storage.Delete(ctx, path); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, path)); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestLocalStorageExists(t *testing.T) {
	tmpDir := t.TempDir()
	storage := service.NewLocalStorage(tmpDir)

	ctx := context.Background()
	content := []byte("exists test")
	path, _ := storage.Save(ctx, "exists.txt", bytes.NewReader(content), int64(len(content)))

	exists, err := storage.Exists(ctx, path)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("file should exist")
	}

	// Check non-existent file
	exists, err = storage.Exists(ctx, "no-such-file.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("file should not exist")
	}
}

func TestLocalStorageURL(t *testing.T) {
	storage := service.NewLocalStorage("/uploads")
	url := storage.URL("images/photo.jpg")
	if url != "/uploads/images/photo.jpg" {
		t.Errorf("unexpected URL: %s", url)
	}
}

func TestLocalStorageSaveSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	storage := service.NewLocalStorage(tmpDir)

	ctx := context.Background()
	content := []byte("nested file")
	path, err := storage.Save(ctx, "sub/dir/file.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Save to subdirectory failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, path)); err != nil {
		t.Fatalf("file not found on disk: %v", err)
	}
}
