package repository

import (
	"context"

	"blotting-consultancy/internal/model"
)

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	// Create creates a new category
	Create(ctx context.Context, category *model.Category) error

	// FindByID finds a category by ID
	FindByID(ctx context.Context, id uint) (*model.Category, error)

	// FindBySlug finds a category by slug
	FindBySlug(ctx context.Context, slug string) (*model.Category, error)

	// Update updates a category
	Update(ctx context.Context, category *model.Category) error

	// Delete deletes a category by ID
	Delete(ctx context.Context, id uint) error

	// List returns all categories
	List(ctx context.Context) ([]*model.Category, error)

	// ListTree returns all categories as a tree structure (root items with Children populated)
	ListTree(ctx context.Context) ([]*model.Category, error)

	// ListByParentID returns categories filtered by parent_id (nil for root items)
	ListByParentID(ctx context.Context, parentID *uint) ([]*model.Category, error)

	// FindByIDs returns categories matching the given IDs
	FindByIDs(ctx context.Context, ids []uint) ([]model.Category, error)
}
