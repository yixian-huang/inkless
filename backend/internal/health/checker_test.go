package health

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestChecker_RunAll_AllHealthy(t *testing.T) {
	c := NewChecker("1.0.0")
	c.AddCheck("test1", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})
	c.AddCheck("test2", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})

	resp := c.RunAll(context.Background())
	assert.Equal(t, StatusHealthy, resp.Status)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.Len(t, resp.Checks, 2)
}

func TestChecker_RunAll_Degraded(t *testing.T) {
	c := NewChecker("1.0.0")
	c.AddCheck("healthy", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})
	c.AddCheck("degraded", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusDegraded}
	})

	resp := c.RunAll(context.Background())
	assert.Equal(t, StatusDegraded, resp.Status)
}

func TestChecker_RunAll_Unhealthy(t *testing.T) {
	c := NewChecker("1.0.0")
	c.AddCheck("healthy", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})
	c.AddCheck("degraded", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusDegraded}
	})
	c.AddCheck("unhealthy", func(_ context.Context) CheckResult {
		return CheckResult{Status: StatusUnhealthy}
	})

	resp := c.RunAll(context.Background())
	assert.Equal(t, StatusUnhealthy, resp.Status)
}

func TestChecker_RunAll_Empty(t *testing.T) {
	c := NewChecker("1.0.0")
	resp := c.RunAll(context.Background())
	assert.Equal(t, StatusHealthy, resp.Status)
	assert.Empty(t, resp.Checks)
}

func TestDatabaseCheck_Healthy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)

	check := DatabaseCheck(db)
	result := check(context.Background())

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "sqlite", result.Details["type"])
	assert.NotNil(t, result.Details["latencyMs"])
	assert.NotEmpty(t, result.Details["version"])
}

func TestStorageCheck_Healthy(t *testing.T) {
	tmpDir := t.TempDir()

	check := StorageCheck(tmpDir)
	result := check(context.Background())

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, true, result.Details["writable"])
	assert.Equal(t, "local", result.Details["provider"])

	// Verify tmp file was cleaned up
	_, err := os.Stat(filepath.Join(tmpDir, ".health-check-tmp"))
	assert.True(t, os.IsNotExist(err))
}

func TestStorageCheck_NonexistentCreatable(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "subdir")

	check := StorageCheck(tmpDir)
	result := check(context.Background())

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, true, result.Details["writable"])
}

func TestMemoryCheck_Healthy(t *testing.T) {
	check := MemoryCheck(10000) // 10 GB threshold, should always be healthy
	result := check(context.Background())

	assert.Equal(t, StatusHealthy, result.Status)
	assert.NotNil(t, result.Details["usedMB"])
	assert.Equal(t, float64(10000), result.Details["thresholdMB"])
}

func TestMemoryCheck_Degraded(t *testing.T) {
	check := MemoryCheck(0.0001) // Impossibly low threshold
	result := check(context.Background())

	assert.Equal(t, StatusDegraded, result.Status)
}

// mockSchedulerInfo implements SchedulerInfo for testing.
type mockSchedulerInfo struct {
	running bool
	enabled bool
	next    time.Time
}

func (m *mockSchedulerInfo) IsRunning() bool   { return m.running }
func (m *mockSchedulerInfo) Enabled() bool      { return m.enabled }
func (m *mockSchedulerInfo) NextRun() time.Time { return m.next }

func TestSchedulerCheck_Healthy(t *testing.T) {
	info := &mockSchedulerInfo{
		running: true,
		enabled: true,
		next:    time.Now().Add(time.Hour),
	}

	check := SchedulerCheckFunc(info)
	result := check(context.Background())

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, true, result.Details["schedulerRunning"])
	assert.NotEmpty(t, result.Details["nextBackupAt"])
}

func TestSchedulerCheck_Disabled(t *testing.T) {
	info := &mockSchedulerInfo{
		running: false,
		enabled: false,
	}

	check := SchedulerCheckFunc(info)
	result := check(context.Background())

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, false, result.Details["schedulerEnabled"])
}

func TestSchedulerCheck_Degraded(t *testing.T) {
	info := &mockSchedulerInfo{
		running: false,
		enabled: true,
	}

	check := SchedulerCheckFunc(info)
	result := check(context.Background())

	assert.Equal(t, StatusDegraded, result.Status)
}
