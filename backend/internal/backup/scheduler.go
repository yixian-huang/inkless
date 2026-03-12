package backup

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"blotting-consultancy/internal/model"

	"github.com/robfig/cron/v3"
)

// Scheduler manages automatic backup scheduling using cron expressions.
type Scheduler struct {
	cron     *cron.Cron
	service  BackupRunner
	logger   *slog.Logger
	schedule string
	enabled  bool

	mu        sync.RWMutex
	running   bool
	lastRunAt time.Time
}

// BackupRunner is the interface needed by the scheduler to trigger backups.
type BackupRunner interface {
	RunBackup(ctx context.Context) (*model.BackupRecord, error)
}

// NewScheduler creates a new backup scheduler.
// schedule is a standard 5-field cron expression (minute hour day month weekday).
func NewScheduler(svc BackupRunner, schedule string, enabled bool) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		service:  svc,
		logger:   slog.Default(),
		schedule: schedule,
		enabled:  enabled,
	}
}

// Start begins the cron scheduler. Returns an error if the schedule is invalid.
func (s *Scheduler) Start() error {
	if !s.enabled {
		s.logger.Info("Backup scheduler disabled")
		return nil
	}

	_, err := s.cron.AddFunc(s.schedule, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		s.logger.Info("Starting scheduled backup")
		record, err := s.service.RunBackup(ctx)
		if err != nil {
			s.logger.Error("Scheduled backup failed", "error", err)
			return
		}

		s.mu.Lock()
		s.lastRunAt = time.Now()
		s.mu.Unlock()

		s.logger.Info("Scheduled backup completed", "filename", record.Filename, "size", record.Size)
	})
	if err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", s.schedule, err)
	}

	s.cron.Start()
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Backup scheduler started", "schedule", s.schedule)
	return nil
}

// Stop halts the cron scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// NextRun returns the next scheduled run time.
func (s *Scheduler) NextRun() time.Time {
	entries := s.cron.Entries()
	if len(entries) > 0 {
		return entries[0].Next
	}
	return time.Time{}
}

// IsRunning returns whether the scheduler is currently active.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// LastRunAt returns the time of the last successful backup triggered by the scheduler.
func (s *Scheduler) LastRunAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRunAt
}

// Schedule returns the configured cron schedule string.
func (s *Scheduler) Schedule() string {
	return s.schedule
}

// Enabled returns whether the scheduler is configured to be enabled.
func (s *Scheduler) Enabled() bool {
	return s.enabled
}
