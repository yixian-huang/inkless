package repository

import (
	"blotting-consultancy/internal/model"
	"context"
	"time"
)

type UnifiedPageRepository interface {
	Create(ctx context.Context, page *model.UnifiedPage) error
	Update(ctx context.Context, page *model.UnifiedPage) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*model.UnifiedPage, error)
	FindBySlug(ctx context.Context, slug string) (*model.UnifiedPage, error)
	List(ctx context.Context, status string, mode string, parentID *uint) ([]*model.UnifiedPage, error)
	ListPublished(ctx context.Context) ([]*model.UnifiedPage, error)
	UpdateDraft(ctx context.Context, id uint, expectedVersion int, draftConfig model.JSONMap) (int, error)
	UpdatePublished(ctx context.Context, id uint, publishedConfig model.JSONMap, publishedVersion int, publishedAt time.Time) error
	UpdateRollback(ctx context.Context, id uint, draftConfig model.JSONMap, draftVersion int, publishedConfig model.JSONMap, publishedVersion int, publishedAt time.Time) error
	ClearPublished(ctx context.Context, id uint) error
	UpdateSortOrder(ctx context.Context, id uint, sortOrder int) error
}
