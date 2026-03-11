package repository

import (
	"context"

	"blotting-consultancy/internal/model"
)

// TagRepository defines the interface for tag data access
type TagRepository interface {
	// Create creates a new tag
	Create(ctx context.Context, tag *model.Tag) error

	// FindByID finds a tag by ID
	FindByID(ctx context.Context, id uint) (*model.Tag, error)

	// FindBySlug finds a tag by slug
	FindBySlug(ctx context.Context, slug string) (*model.Tag, error)

	// Update updates a tag
	Update(ctx context.Context, tag *model.Tag) error

	// Delete deletes a tag by ID
	Delete(ctx context.Context, id uint) error

	// List returns all tags
	List(ctx context.Context) ([]*model.Tag, error)

	// FindByIDs returns tags matching the given IDs
	FindByIDs(ctx context.Context, ids []uint) ([]model.Tag, error)
}
