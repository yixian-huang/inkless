package repository

import (
	"context"
	"errors"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// GormTagRepository implements TagRepository using GORM
type GormTagRepository struct {
	db *gorm.DB
}

// NewGormTagRepository creates a new GormTagRepository
func NewGormTagRepository(db *gorm.DB) TagRepository {
	return &GormTagRepository{db: db}
}

// Create creates a new tag
func (r *GormTagRepository) Create(ctx context.Context, tag *model.Tag) error {
	if err := tag.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(tag).Error
}

// FindByID finds a tag by ID
func (r *GormTagRepository) FindByID(ctx context.Context, id uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// FindBySlug finds a tag by slug
func (r *GormTagRepository) FindBySlug(ctx context.Context, slug string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// Update updates a tag
func (r *GormTagRepository) Update(ctx context.Context, tag *model.Tag) error {
	if err := tag.Validate(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Save(tag)
	return result.Error
}

// Delete deletes a tag by ID
func (r *GormTagRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Tag{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("tag not found")
	}
	return nil
}

// FindByIDs returns tags matching the given IDs
func (r *GormTagRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.Tag, error) {
	var items []model.Tag
	if len(ids) == 0 {
		return items, nil
	}
	if err := r.db.WithContext(ctx).Where("id IN (?)", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// List returns all tags
func (r *GormTagRepository) List(ctx context.Context) ([]*model.Tag, error) {
	var items []*model.Tag
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
