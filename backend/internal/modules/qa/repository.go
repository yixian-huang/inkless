package qa

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// QALogRepository defines the interface for Q&A log data access.
type QALogRepository interface {
	Create(ctx context.Context, log *QALog) error
	FindByID(ctx context.Context, id uint) (*QALog, error)
	List(ctx context.Context, offset, limit int) ([]*QALog, int64, error)
	UpdateRating(ctx context.Context, id uint, rating QAFeedback) error
}

// gormQALogRepository implements QALogRepository using GORM.
type gormQALogRepository struct {
	db *gorm.DB
}

// newGormQALogRepository creates a new gormQALogRepository.
func newGormQALogRepository(db *gorm.DB) QALogRepository {
	return &gormQALogRepository{db: db}
}

// Create creates a new Q&A log entry.
func (r *gormQALogRepository) Create(ctx context.Context, log *QALog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// FindByID finds a Q&A log entry by ID.
func (r *gormQALogRepository) FindByID(ctx context.Context, id uint) (*QALog, error) {
	var log QALog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("qa log not found")
		}
		return nil, err
	}
	return &log, nil
}

// List returns paginated Q&A log entries, ordered by created_at DESC.
func (r *gormQALogRepository) List(ctx context.Context, offset, limit int) ([]*QALog, int64, error) {
	var logs []*QALog
	var total int64

	query := r.db.WithContext(ctx).Model(&QALog{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// UpdateRating updates the rating of a Q&A log entry.
func (r *gormQALogRepository) UpdateRating(ctx context.Context, id uint, rating QAFeedback) error {
	result := r.db.WithContext(ctx).
		Model(&QALog{}).
		Where("id = ?", id).
		Update("rating", rating)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("qa log not found")
	}
	return nil
}
