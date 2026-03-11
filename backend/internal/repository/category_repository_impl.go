package repository

import (
	"context"
	"errors"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// GormCategoryRepository implements CategoryRepository using GORM
type GormCategoryRepository struct {
	db *gorm.DB
}

// NewGormCategoryRepository creates a new GormCategoryRepository
func NewGormCategoryRepository(db *gorm.DB) CategoryRepository {
	return &GormCategoryRepository{db: db}
}

// Create creates a new category
func (r *GormCategoryRepository) Create(ctx context.Context, category *model.Category) error {
	if err := category.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(category).Error
}

// FindByID finds a category by ID
func (r *GormCategoryRepository) FindByID(ctx context.Context, id uint) (*model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).First(&category, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
}

// Update updates a category
func (r *GormCategoryRepository) Update(ctx context.Context, category *model.Category) error {
	if err := category.Validate(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Save(category)
	return result.Error
}

// Delete deletes a category by ID
func (r *GormCategoryRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Category{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("category not found")
	}
	return nil
}

// FindBySlug finds a category by slug
func (r *GormCategoryRepository) FindBySlug(ctx context.Context, slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
}

// List returns all categories
func (r *GormCategoryRepository) List(ctx context.Context) ([]*model.Category, error) {
	var items []*model.Category
	if err := r.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListTree returns all categories as a tree structure
func (r *GormCategoryRepository) ListTree(ctx context.Context) ([]*model.Category, error) {
	var all []*model.Category
	if err := r.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&all).Error; err != nil {
		return nil, err
	}

	// Build parent-children tree in Go
	byID := make(map[uint]*model.Category)
	for _, c := range all {
		byID[c.ID] = c
	}

	var roots []*model.Category
	for _, c := range all {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else if parent, ok := byID[*c.ParentID]; ok {
			parent.Children = append(parent.Children, *c)
		}
	}
	return roots, nil
}

// FindByIDs returns categories matching the given IDs
func (r *GormCategoryRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.Category, error) {
	var items []model.Category
	if len(ids) == 0 {
		return items, nil
	}
	if err := r.db.WithContext(ctx).Where("id IN (?)", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByParentID returns categories filtered by parent_id (nil for root items)
func (r *GormCategoryRepository) ListByParentID(ctx context.Context, parentID *uint) ([]*model.Category, error) {
	var items []*model.Category
	query := r.db.WithContext(ctx).Order("sort_order ASC, id ASC")
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
