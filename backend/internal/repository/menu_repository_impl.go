package repository

import (
	"context"
	"errors"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// GormMenuRepository implements MenuRepository using GORM
type GormMenuRepository struct {
	db *gorm.DB
}

// NewGormMenuRepository creates a new GormMenuRepository
func NewGormMenuRepository(db *gorm.DB) MenuRepository {
	return &GormMenuRepository{db: db}
}

// CreateGroup creates a new menu group
func (r *GormMenuRepository) CreateGroup(ctx context.Context, group *model.MenuGroup) error {
	if err := group.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(group).Error
}

// FindGroupByID finds a menu group by ID with items preloaded ordered by sort_order
func (r *GormMenuRepository) FindGroupByID(ctx context.Context, id uint) (*model.MenuGroup, error) {
	var group model.MenuGroup
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, id ASC")
		}).
		First(&group, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("menu group not found")
		}
		return nil, err
	}
	return &group, nil
}

// UpdateGroup updates a menu group
func (r *GormMenuRepository) UpdateGroup(ctx context.Context, group *model.MenuGroup) error {
	if err := group.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(group).Error
}

// DeleteGroup deletes a menu group by ID (and its items)
func (r *GormMenuRepository) DeleteGroup(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete all items in the group first
		if err := tx.Where("group_id = ?", id).Delete(&model.MenuItem{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.MenuGroup{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("menu group not found")
		}
		return nil
	})
}

// ListGroups returns all menu groups (without items)
func (r *GormMenuRepository) ListGroups(ctx context.Context) ([]*model.MenuGroup, error) {
	var groups []*model.MenuGroup
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// FindPrimaryGroup finds the primary menu group with items preloaded
func (r *GormMenuRepository) FindPrimaryGroup(ctx context.Context) (*model.MenuGroup, error) {
	var group model.MenuGroup
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, id ASC")
		}).
		Where("is_primary = ?", true).
		First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("primary menu group not found")
		}
		return nil, err
	}
	return &group, nil
}

// SetPrimary sets a menu group as primary (and unsets all others)
func (r *GormMenuRepository) SetPrimary(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset all primary flags
		if err := tx.Model(&model.MenuGroup{}).Where("1 = 1").Update("is_primary", false).Error; err != nil {
			return err
		}
		// Set the target as primary
		result := tx.Model(&model.MenuGroup{}).Where("id = ?", id).Update("is_primary", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("menu group not found")
		}
		return nil
	})
}

// CreateItem creates a new menu item
func (r *GormMenuRepository) CreateItem(ctx context.Context, item *model.MenuItem) error {
	if err := item.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(item).Error
}

// FindItemByID finds a menu item by ID
func (r *GormMenuRepository) FindItemByID(ctx context.Context, id uint) (*model.MenuItem, error) {
	var item model.MenuItem
	err := r.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("menu item not found")
		}
		return nil, err
	}
	return &item, nil
}

// UpdateItem updates a menu item
func (r *GormMenuRepository) UpdateItem(ctx context.Context, item *model.MenuItem) error {
	if err := item.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(item).Error
}

// DeleteItem deletes a menu item by ID
func (r *GormMenuRepository) DeleteItem(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.MenuItem{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("menu item not found")
	}
	return nil
}

// ListItemsByGroupID returns all menu items for a group ordered by sort_order
func (r *GormMenuRepository) ListItemsByGroupID(ctx context.Context, groupID uint) ([]*model.MenuItem, error) {
	var items []*model.MenuItem
	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("sort_order ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ReorderItems updates sort_order for items by their position in the itemIDs slice
func (r *GormMenuRepository) ReorderItems(ctx context.Context, groupID uint, itemIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, itemID := range itemIDs {
			if err := tx.Model(&model.MenuItem{}).
				Where("id = ? AND group_id = ?", itemID, groupID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
