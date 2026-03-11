package repository

import (
	"context"
	"errors"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// GormArticleRepository implements ArticleRepository using GORM
type GormArticleRepository struct {
	db *gorm.DB
}

// NewGormArticleRepository creates a new GormArticleRepository
func NewGormArticleRepository(db *gorm.DB) ArticleRepository {
	return &GormArticleRepository{db: db}
}

// Create creates a new article
func (r *GormArticleRepository) Create(ctx context.Context, article *model.Article) error {
	if err := article.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(article).Error
}

// FindByID finds an article by ID with Category and Tags preloaded
func (r *GormArticleRepository) FindByID(ctx context.Context, id uint) (*model.Article, error) {
	var article model.Article
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Categories").
		Preload("Tags").
		First(&article, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, err
	}
	return &article, nil
}

// FindBySlug finds an article by slug with Category and Tags preloaded
func (r *GormArticleRepository) FindBySlug(ctx context.Context, slug string) (*model.Article, error) {
	var article model.Article
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Categories").
		Preload("Tags").
		Where("slug = ?", slug).
		First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, err
	}
	return &article, nil
}

// Update updates an article
func (r *GormArticleRepository) Update(ctx context.Context, article *model.Article) error {
	if err := article.Validate(); err != nil {
		return err
	}
	// Replace tags association
	if err := r.db.WithContext(ctx).Model(article).Association("Tags").Replace(article.Tags); err != nil {
		return err
	}
	// Replace categories association
	if err := r.db.WithContext(ctx).Model(article).Association("Categories").Replace(article.Categories); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(article).Error
}

// Delete deletes an article by ID
func (r *GormArticleRepository) Delete(ctx context.Context, id uint) error {
	// Clear associations first
	article := &model.Article{ID: id}
	if err := r.db.WithContext(ctx).Model(article).Association("Tags").Clear(); err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(article).Association("Categories").Clear(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Delete(&model.Article{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("article not found")
	}
	return nil
}

// List returns a paginated list of articles with optional filters
func (r *GormArticleRepository) List(ctx context.Context, offset, limit int, status string, categoryID *uint, tagID *uint) ([]*model.Article, int64, error) {
	var items []*model.Article
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Article{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}
	if tagID != nil {
		query = query.Where("id IN (SELECT article_id FROM article_tags WHERE tag_id = ?)", *tagID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Categories").
		Preload("Tags").
		Scopes(buildWhere(status, categoryID, tagID)).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// publishedScope returns a GORM scope that applies the published article filters
func publishedScope(categorySlug, tagSlug string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = db.Where("status = ?", model.ArticleStatusPublished).
			Where("visibility = 'public' OR visibility = '' OR visibility IS NULL")
		if categorySlug != "" {
			db = db.Where("category_id IN (SELECT id FROM categories WHERE slug = ?) OR id IN (SELECT article_id FROM article_categories WHERE category_id IN (SELECT id FROM categories WHERE slug = ?))", categorySlug, categorySlug)
		}
		if tagSlug != "" {
			db = db.Where("id IN (SELECT article_id FROM article_tags WHERE tag_id IN (SELECT id FROM tags WHERE slug = ?))", tagSlug)
		}
		return db
	}
}

// ListPublished returns a paginated list of published articles
func (r *GormArticleRepository) ListPublished(ctx context.Context, offset, limit int, categorySlug string, tagSlug string) ([]*model.Article, int64, error) {
	var items []*model.Article
	var total int64

	scope := publishedScope(categorySlug, tagSlug)

	if err := r.db.WithContext(ctx).Model(&model.Article{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Categories").
		Preload("Tags").
		Scopes(scope).
		Offset(offset).
		Limit(limit).
		Order("pinned DESC, published_at DESC, created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// buildWhere constructs a GORM scope function for the List query filters
func buildWhere(status string, categoryID *uint, tagID *uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if status != "" {
			db = db.Where("status = ?", status)
		}
		if categoryID != nil {
			db = db.Where("category_id = ?", *categoryID)
		}
		if tagID != nil {
			db = db.Where("id IN (SELECT article_id FROM article_tags WHERE tag_id = ?)", *tagID)
		}
		return db
	}
}
