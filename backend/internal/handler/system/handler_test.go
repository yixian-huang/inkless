package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)

	// Create minimal tables for counting
	sqlDB, err := db.DB()
	require.NoError(t, err)

	for _, table := range []string{"articles", "pages", "media", "users"} {
		_, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS " + table + " (id INTEGER PRIMARY KEY)")
		require.NoError(t, err)
	}

	return db
}

func TestGetStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)

	// Create a temp upload dir with a small file
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)
	require.NoError(t, err)

	handler := NewHandler(db, tmpDir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/system/status", nil)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Runtime checks
	assert.Equal(t, runtime.Version(), resp.Runtime.GoVersion)
	assert.Equal(t, runtime.GOOS, resp.Runtime.OS)
	assert.Equal(t, runtime.GOARCH, resp.Runtime.Arch)
	assert.Equal(t, runtime.NumCPU(), resp.Runtime.CPUCount)
	assert.Greater(t, resp.Runtime.Goroutines, 0)
	assert.GreaterOrEqual(t, resp.Runtime.UptimeSec, int64(0))

	// Memory checks
	assert.Greater(t, resp.Memory.AllocMB, 0.0)
	assert.Greater(t, resp.Memory.SysMB, 0.0)

	// Database checks
	assert.Equal(t, "sqlite", resp.Database.Type)

	// Storage checks
	assert.Greater(t, resp.Storage.UploadDirSizeMB, 0.0)

	// Content counts (all zero in test)
	assert.Equal(t, int64(0), resp.Content.Articles)
	assert.Equal(t, int64(0), resp.Content.Pages)
}

func TestDirSizeMB(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty dir
	size := dirSizeMB(tmpDir)
	assert.Equal(t, 0.0, size)

	// Write 1024 bytes
	err := os.WriteFile(filepath.Join(tmpDir, "file.bin"), make([]byte, 1024), 0644)
	require.NoError(t, err)

	size = dirSizeMB(tmpDir)
	assert.InDelta(t, 0.001, size, 0.001)

	// Non-existent dir
	size = dirSizeMB("/nonexistent/path/xyz")
	assert.Equal(t, 0.0, size)
}
