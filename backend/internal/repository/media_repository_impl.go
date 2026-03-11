package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// GormMediaRepository implements MediaRepository using GORM
type GormMediaRepository struct {
	db *gorm.DB
}

// NewGormMediaRepository creates a new GormMediaRepository
func NewGormMediaRepository(db *gorm.DB) MediaRepository {
	return &GormMediaRepository{db: db}
}

// Create creates a new media record
func (r *GormMediaRepository) Create(ctx context.Context, media *model.Media) error {
	if err := media.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(media).Error
}

// FindByID finds a media record by ID
func (r *GormMediaRepository) FindByID(ctx context.Context, id uint) (*model.Media, error) {
	var media model.Media
	err := r.db.WithContext(ctx).First(&media, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("media not found")
		}
		return nil, err
	}
	return &media, nil
}

// List returns a paginated list of media records ordered by creation time (newest first).
// When mimePrefix is non-empty, only records whose mime_type starts with that prefix are returned.
func (r *GormMediaRepository) List(ctx context.Context, offset, limit int, mimePrefix string) ([]*model.Media, int64, error) {
	var items []*model.Media
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Media{})
	if mimePrefix != "" {
		query = query.Where("mime_type LIKE ?", mimePrefix+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// Delete deletes a media record by ID
func (r *GormMediaRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Media{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("media not found")
	}
	return nil
}

// Update updates an existing media record
func (r *GormMediaRepository) Update(ctx context.Context, media *model.Media) error {
	if err := media.Validate(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Save(media)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("media not found")
	}
	return nil
}

// FindUsages searches for references to a media URL across articles, pages, and content documents
func (r *GormMediaRepository) FindUsages(ctx context.Context, mediaURL string) ([]MediaUsage, error) {
	var usages []MediaUsage
	pattern := "%" + mediaURL + "%"

	// Search articles
	var articles []struct {
		ID         uint
		ZhTitle    string
		CoverImage string
		OgImage    string
		ZhBody     string
		EnBody     string
	}
	if err := r.db.WithContext(ctx).
		Table("articles").
		Select("id, zh_title, cover_image, og_image, zh_body, en_body").
		Where("cover_image LIKE ? OR og_image LIKE ? OR zh_body LIKE ? OR en_body LIKE ?",
			pattern, pattern, pattern, pattern).
		Find(&articles).Error; err != nil {
		return nil, err
	}

	for _, a := range articles {
		var fields []string
		if strings.Contains(a.CoverImage, mediaURL) {
			fields = append(fields, "封面图")
		}
		if strings.Contains(a.OgImage, mediaURL) {
			fields = append(fields, "OG图片")
		}
		if strings.Contains(a.ZhBody, mediaURL) {
			fields = append(fields, "中文正文")
		}
		if strings.Contains(a.EnBody, mediaURL) {
			fields = append(fields, "英文正文")
		}
		usages = append(usages, MediaUsage{
			Type:  "article",
			ID:    fmt.Sprintf("%d", a.ID),
			Title: a.ZhTitle,
			Field: strings.Join(fields, ", "),
		})
	}

	// Search pages (has soft delete)
	var pages []struct {
		ID   uint
		Slug string
	}
	if err := r.db.WithContext(ctx).
		Table("pages").
		Select("id, slug").
		Where("deleted_at IS NULL AND CAST(config AS TEXT) LIKE ?", pattern).
		Find(&pages).Error; err != nil {
		return nil, err
	}

	for _, p := range pages {
		usages = append(usages, MediaUsage{
			Type:  "page",
			ID:    fmt.Sprintf("%d", p.ID),
			Title: p.Slug,
			Field: "页面配置",
		})
	}

	// Search content documents
	var docs []struct {
		PageKey string
	}
	if err := r.db.WithContext(ctx).
		Table("content_documents").
		Select("page_key").
		Where("CAST(draft_config AS TEXT) LIKE ? OR CAST(published_config AS TEXT) LIKE ?",
			pattern, pattern).
		Find(&docs).Error; err != nil {
		return nil, err
	}

	for _, d := range docs {
		usages = append(usages, MediaUsage{
			Type:  "content_document",
			ID:    d.PageKey,
			Title: d.PageKey,
			Field: "内容配置",
		})
	}

	return usages, nil
}
