package service

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type SchedulerService struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSchedulerService(db *gorm.DB) *SchedulerService {
	return &SchedulerService{
		db:     db,
		logger: slog.Default(),
	}
}

func (s *SchedulerService) PublishOverdue() (int, error) {
	now := time.Now()
	total := 0

	result := s.db.Table("articles").
		Where("status = ? AND scheduled_at <= ?", "scheduled", now).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	total += int(result.RowsAffected)

	result = s.db.Table("pages").
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
		for range ticker.C {
			if _, err := s.PublishOverdue(); err != nil {
				s.logger.Error("Scheduler error", "error", err)
			}
		}
	}()
	s.logger.Info("Scheduler started (checking every 1 minute)")
}
