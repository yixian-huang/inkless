package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"

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

func (r *GormArticleRepository) UpdateScheduledPublication(
	ctx context.Context,
	article *model.Article,
	expectedUpdatedAt time.Time,
) error {
	if err := article.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		article.UpdatedAt = time.Now()
		result := tx.Model(&model.Article{}).
			Where("id = ? AND updated_at = ?", article.ID, expectedUpdatedAt).
			Select("*").
			Omit("id", "created_at", "Category", "Categories", "Tags").
			Updates(article)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrArticleVersionConflict
		}
		if err := tx.Model(article).Association("Tags").Replace(article.Tags); err != nil {
			return err
		}
		if err := tx.Model(article).Association("Categories").Replace(article.Categories); err != nil {
			return err
		}
		return nil
	})
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

	scope := buildWhere(status, categoryID, tagID)

	if err := r.db.WithContext(ctx).Model(&model.Article{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Select(articleListSelectColumns).
		Preload("Category").
		Preload("Categories").
		Preload("Tags").
		Scopes(scope).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// articleListSelectColumns omits large text bodies from admin list queries.
// Column names are snake_case as stored by GORM.
const articleListSelectColumns = "id, slug, status, zh_title, en_title, cover_image, " +
	"zh_seo_title, en_seo_title, zh_meta_description, en_meta_description, og_image, " +
	"category_id, author, auto_summary, allow_comments, pinned, visibility, " +
	"scheduled_at, published_at, created_at, updated_at"

// articlePublicListSelectColumns includes bodies so PublicList can build short
// plain-text excerpts without a second query per row.
const articlePublicListSelectColumns = articleListSelectColumns + ", zh_body, en_body"

// publishedScope returns a GORM scope that applies the published article filters
func publishedScope(categorySlug, tagSlug string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = db.Where("status = ?", model.ArticleStatusPublished).
			Where("visibility = 'public' OR visibility = '' OR visibility IS NULL")
		if categorySlug != "" {
			db = db.Where(
				"category_id IN (SELECT id FROM categories WHERE slug = ?) OR id IN (SELECT article_id FROM article_categories ac JOIN categories c ON ac.category_id = c.id WHERE c.slug = ?)",
				categorySlug, categorySlug,
			)
		}
		if tagSlug != "" {
			db = db.Where(
				"id IN (SELECT article_id FROM article_tags at JOIN tags t ON at.tag_id = t.id WHERE t.slug = ?)",
				tagSlug,
			)
		}
		return db
	}
}

// ListPublished returns a paginated list of published articles.
// Includes zh_body/en_body for public list excerpts (handlers should truncate).
func (r *GormArticleRepository) ListPublished(ctx context.Context, offset, limit int, categorySlug string, tagSlug string) ([]*model.Article, int64, error) {
	var items []*model.Article
	var total int64

	scope := publishedScope(categorySlug, tagSlug)

	if err := r.db.WithContext(ctx).Model(&model.Article{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Select(articlePublicListSelectColumns).
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

// Count returns article totals, optionally filtered by status.
func (r *GormArticleRepository) Count(ctx context.Context, status string) (int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&model.Article{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
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
