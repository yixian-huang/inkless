package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
)

var ErrArticleVersionConflict = errors.New("article version conflict")

// ArticleRepository defines the interface for article data access
type ArticleRepository interface {
	// Create creates a new article
	Create(ctx context.Context, article *model.Article) error

	// FindByID finds an article by ID with Category and Tags preloaded
	FindByID(ctx context.Context, id uint) (*model.Article, error)

	// FindBySlug finds an article by slug with Category and Tags preloaded
	FindBySlug(ctx context.Context, slug string) (*model.Article, error)

	// Update updates an article
	Update(ctx context.Context, article *model.Article) error
	UpdateScheduledPublication(ctx context.Context, article *model.Article, expectedUpdatedAt time.Time) error

	// Delete deletes an article by ID
	Delete(ctx context.Context, id uint) error

	// List returns a paginated list of articles with optional filters.
	// Body fields (zh_body/en_body) are omitted for list performance.
	List(ctx context.Context, offset, limit int, status string, categoryID *uint, tagID *uint) ([]*model.Article, int64, error)

	// ListPublished returns a paginated list of published articles with optional filters.
	// Includes body fields so public list can derive short excerpts; full HTML
	// content for reading still comes from FindBySlug.
	ListPublished(ctx context.Context, offset, limit int, categorySlug string, tagSlug string) ([]*model.Article, int64, error)

	// Count returns the number of articles, optionally filtered by status (empty = all).
	Count(ctx context.Context, status string) (int64, error)
}
