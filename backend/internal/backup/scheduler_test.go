package backup

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"blotting-consultancy/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBackupRunner is a test double that records calls.
type mockBackupRunner struct {
	callCount atomic.Int32
	shouldErr error
}

func (m *mockBackupRunner) RunBackup(_ context.Context) (*model.BackupRecord, error) {
	m.callCount.Add(1)
	if m.shouldErr != nil {
		return nil, m.shouldErr
	}
	return &model.BackupRecord{
		Filename: "backup-test.db.gz",
		Size:     1234,
	}, nil
}

func TestNewScheduler(t *testing.T) {
	mock := &mockBackupRunner{}
	s := NewScheduler(mock, "0 2 * * *", true)

	assert.NotNil(t, s)
	assert.Equal(t, "0 2 * * *", s.Schedule())
	assert.True(t, s.Enabled())
	assert.False(t, s.IsRunning())
}

func TestScheduler_StartDisabled(t *testing.T) {
	mock := &mockBackupRunner{}
	s := NewScheduler(mock, "0 2 * * *", false)

	err := s.Start()
	assert.NoError(t, err)
	assert.False(t, s.IsRunning())
}

func TestScheduler_StartInvalidSchedule(t *testing.T) {
	mock := &mockBackupRunner{}
	s := NewScheduler(mock, "invalid-cron", true)

	err := s.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron schedule")
}

func TestScheduler_StartAndStop(t *testing.T) {
	mock := &mockBackupRunner{}
	// Every second for testing
	s := NewScheduler(mock, "* * * * *", true)

	err := s.Start()
	require.NoError(t, err)
	assert.True(t, s.IsRunning())

	// NextRun should be in the future
	next := s.NextRun()
	assert.False(t, next.IsZero())
	assert.True(t, next.After(time.Now().Add(-time.Second)))

	s.Stop()
	assert.False(t, s.IsRunning())
}

func TestScheduler_NextRunEmpty(t *testing.T) {
	mock := &mockBackupRunner{}
	s := NewScheduler(mock, "0 2 * * *", false)

	// Not started, no entries
	next := s.NextRun()
	assert.True(t, next.IsZero())
}

func TestScheduler_LastRunAt(t *testing.T) {
	mock := &mockBackupRunner{}
	s := NewScheduler(mock, "0 2 * * *", true)

	// Before any run
	assert.True(t, s.LastRunAt().IsZero())
}
