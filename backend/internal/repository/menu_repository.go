package repository

import (
	"context"

	"blotting-consultancy/internal/model"
)

// MenuRepository defines the interface for menu data access
type MenuRepository interface {
	// CreateGroup creates a new menu group
	CreateGroup(ctx context.Context, group *model.MenuGroup) error

	// FindGroupByID finds a menu group by ID with items preloaded
	FindGroupByID(ctx context.Context, id uint) (*model.MenuGroup, error)

	// UpdateGroup updates a menu group
	UpdateGroup(ctx context.Context, group *model.MenuGroup) error

	// DeleteGroup deletes a menu group by ID
	DeleteGroup(ctx context.Context, id uint) error

	// ListGroups returns all menu groups
	ListGroups(ctx context.Context) ([]*model.MenuGroup, error)

	// FindPrimaryGroup finds the primary menu group with items preloaded
	FindPrimaryGroup(ctx context.Context) (*model.MenuGroup, error)

	// SetPrimary sets a menu group as primary (and unsets all others)
	SetPrimary(ctx context.Context, id uint) error

	// CreateItem creates a new menu item
	CreateItem(ctx context.Context, item *model.MenuItem) error

	// FindItemByID finds a menu item by ID
	FindItemByID(ctx context.Context, id uint) (*model.MenuItem, error)

	// UpdateItem updates a menu item
	UpdateItem(ctx context.Context, item *model.MenuItem) error

	// DeleteItem deletes a menu item by ID
	DeleteItem(ctx context.Context, id uint) error

	// ListItemsByGroupID returns all menu items for a group
	ListItemsByGroupID(ctx context.Context, groupID uint) ([]*model.MenuItem, error)

	// ReorderItems updates sort_order for items in a group
	ReorderItems(ctx context.Context, groupID uint, itemIDs []uint) error
}
