package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"blotting-consultancy/internal/model"
	"gorm.io/gorm"
)

type GormUnifiedPageRepository struct {
	db *gorm.DB
}

func NewGormUnifiedPageRepository(db *gorm.DB) UnifiedPageRepository {
	return &GormUnifiedPageRepository{db: db}
}

func (r *GormUnifiedPageRepository) Create(ctx context.Context, page *model.UnifiedPage) error {
	return r.db.WithContext(ctx).Create(page).Error
}

func (r *GormUnifiedPageRepository) Update(ctx context.Context, page *model.UnifiedPage) error {
	return r.db.WithContext(ctx).Save(page).Error
}

func (r *GormUnifiedPageRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.UnifiedPage{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormUnifiedPageRepository) FindByID(ctx context.Context, id uint) (*model.UnifiedPage, error) {
	var page model.UnifiedPage
	if err := r.db.WithContext(ctx).First(&page, id).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *GormUnifiedPageRepository) FindBySlug(ctx context.Context, slug string) (*model.UnifiedPage, error) {
	var page model.UnifiedPage
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&page).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *GormUnifiedPageRepository) List(ctx context.Context, status string, mode string, parentID *uint) ([]*model.UnifiedPage, error) {
	q := r.db.WithContext(ctx).Model(&model.UnifiedPage{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if mode != "" {
		q = q.Where("mode = ?", mode)
	}
	if parentID != nil {
		q = q.Where("parent_id = ?", *parentID)
	}
	var pages []*model.UnifiedPage
	err := q.Order("sort_order ASC, created_at DESC").Find(&pages).Error
	return pages, err
}

func (r *GormUnifiedPageRepository) ListPublished(ctx context.Context) ([]*model.UnifiedPage, error) {
	var pages []*model.UnifiedPage
	err := r.db.WithContext(ctx).Where("status = ?", "published").Order("sort_order ASC, created_at DESC").Find(&pages).Error
	return pages, err
}

func (r *GormUnifiedPageRepository) UpdateDraft(ctx context.Context, id uint, expectedVersion int, draftConfig model.JSONMap) (int, error) {
	result := r.db.WithContext(ctx).Table("unified_pages").Where("id = ? AND draft_version = ?", id, expectedVersion).Updates(map[string]interface{}{
		"draft_config":  draftConfig,
		"draft_version": gorm.Expr("draft_version + 1"),
	})
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, errors.New("draft version conflict or page not found")
	}
	var page model.UnifiedPage
	if err := r.db.WithContext(ctx).Select("draft_version").First(&page, id).Error; err != nil {
		return 0, fmt.Errorf("fetch new version: %w", err)
	}
	return page.DraftVersion, nil
}

func (r *GormUnifiedPageRepository) UpdatePublished(
	ctx context.Context,
	id uint,
	publishedConfig model.JSONMap,
	publishedVersion int,
	publishedAt time.Time,
) error {
	result := r.db.WithContext(ctx).Table("unified_pages").Where("id = ?", id).Updates(map[string]interface{}{
		"published_config":  publishedConfig,
		"published_version": publishedVersion,
		"status":            "published",
		"published_at":      publishedAt,
	})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormUnifiedPageRepository) UpdateRollback(
	ctx context.Context,
	id uint,
	draftConfig model.JSONMap,
	draftVersion int,
	publishedConfig model.JSONMap,
	publishedVersion int,
	publishedAt time.Time,
) error {
	result := r.db.WithContext(ctx).Table("unified_pages").Where("id = ?", id).Updates(map[string]interface{}{
		"draft_config":      draftConfig,
		"draft_version":     draftVersion,
		"published_config":  publishedConfig,
		"published_version": publishedVersion,
		"status":            "published",
		"published_at":      publishedAt,
	})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormUnifiedPageRepository) ClearPublished(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Table("unified_pages").Where("id = ?", id).Updates(map[string]interface{}{
		"published_config": nil,
		"status":           "draft",
	})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormUnifiedPageRepository) UpdateSortOrder(ctx context.Context, id uint, sortOrder int) error {
	result := r.db.WithContext(ctx).Table("unified_pages").Where("id = ?", id).Update("sort_order", sortOrder)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
