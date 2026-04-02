package service

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type SchedulerService struct {
	db     *gorm.DB
	logger *slog.Logger
	done   chan struct{}
}

func NewSchedulerService(db *gorm.DB) *SchedulerService {
	return &SchedulerService{
		db:     db,
		logger: slog.Default(),
		done:   make(chan struct{}),
	}
}

func (s *SchedulerService) PublishOverdue(ctx context.Context) (int, error) {
	now := time.Now()
	total := 0

	result := s.db.WithContext(ctx).Table("articles").
		Where("status = ? AND scheduled_at <= ?", "scheduled", now).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	total += int(result.RowsAffected)

	result = s.db.WithContext(ctx).Table("pages").
		Where("status = ? AND scheduled_at <= ?", "scheduled", now).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": now,
		})
	if result.Error != nil {
		return total, result.Error
	}
	total += int(result.RowsAffected)

	if total > 0 {
		s.logger.Info("Scheduled publishing completed", "count", total)
	}

	return total, nil
}

func (s *SchedulerService) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if _, err := s.PublishOverdue(ctx); err != nil {
					s.logger.Error("Scheduler error", "error", err)
				}
				cancel()
			}
		}
	}()
	s.logger.Info("Scheduler started (checking every 1 minute)")
}

func (s *SchedulerService) Stop() {
	close(s.done)
	s.logger.Info("Scheduler stopped")
}
