package repository

import (
	"context"
	"blotting-consultancy/internal/model"
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
	UpdatePublished(ctx context.Context, id uint, publishedConfig model.JSONMap, publishedVersion int) error
	UpdateSortOrder(ctx context.Context, id uint, sortOrder int) error
}
