package repository

import (
	"context"
	"errors"

	"blotting-consultancy/internal/model"
	"gorm.io/gorm"
)

// GormRoleRepository implements RoleRepository using GORM
type GormRoleRepository struct {
	db        *gorm.DB
	rbacScope legacyRBACScope
}

// NewGormRoleRepository creates a new GormRoleRepository
func NewGormRoleRepository(db *gorm.DB) RoleRepository {
	return &GormRoleRepository{db: db, rbacScope: detectLegacyRBACScope(db)}
}

// Create creates a new role
func (r *GormRoleRepository) Create(ctx context.Context, role *model.RBACRole) error {
	if err := role.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(role).Error
}

// FindByID finds a role by ID, including its permissions
func (r *GormRoleRepository) FindByID(ctx context.Context, id uint) (*model.RBACRole, error) {
	var role model.RBACRole
	err := r.rbacScope.currentRoles(r.db.WithContext(ctx)).
		Preload("Permissions", "resource <> ?", model.LegacyResourceSites).
		First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// FindByName finds a role by name, including its permissions
func (r *GormRoleRepository) FindByName(ctx context.Context, name string) (*model.RBACRole, error) {
	var role model.RBACRole
	err := r.rbacScope.currentRoles(r.db.WithContext(ctx)).
		Preload("Permissions", "resource <> ?", model.LegacyResourceSites).
		Where("name = ?", name).
		First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// Update updates an existing role
func (r *GormRoleRepository) Update(ctx context.Context, role *model.RBACRole) error {
	if err := role.Validate(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Save(role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("role not found")
	}
	return nil
}

// Delete deletes a role by ID
func (r *GormRoleRepository) Delete(ctx context.Context, id uint) error {
	// Use a transaction to clean up join table entries
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete role_permissions entries
		if err := tx.Exec("DELETE FROM role_permissions WHERE rbac_role_id = ?", id).Error; err != nil {
			return err
		}
		// Delete user_roles entries
		if err := r.rbacScope.currentUserRoles(tx).Where("role_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		// Delete the role itself
		result := r.rbacScope.currentRoles(tx).Delete(&model.RBACRole{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("role not found")
		}
		return nil
	})
}

// List returns all roles with their permissions
func (r *GormRoleRepository) List(ctx context.Context) ([]*model.RBACRole, error) {
	var roles []*model.RBACRole
	err := r.rbacScope.currentRoles(r.db.WithContext(ctx)).
		Preload("Permissions", "resource <> ?", model.LegacyResourceSites).
		Order("id ASC").
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// SetPermissions replaces all permissions for a role
func (r *GormRoleRepository) SetPermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := r.rbacScope.currentRoles(tx.Model(&model.RBACRole{})).Where("id = ?", roleID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("role not found")
		}

		// Clear existing permissions
		if err := tx.Exec("DELETE FROM role_permissions WHERE rbac_role_id = ?", roleID).Error; err != nil {
			return err
		}

		// Insert new permissions
		for _, permID := range permissionIDs {
			if err := tx.Exec("INSERT INTO role_permissions (rbac_role_id, permission_id) VALUES (?, ?)", roleID, permID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListPermissions returns all available permissions
func (r *GormRoleRepository) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	var perms []*model.Permission
	err := r.db.WithContext(ctx).
		Where("resource <> ?", model.LegacyResourceSites).
		Order("resource ASC, action ASC").
		Find(&perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// CreatePermission creates a new permission
func (r *GormRoleRepository) CreatePermission(ctx context.Context, perm *model.Permission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

// FindPermissionByResourceAction finds a permission by resource and action
func (r *GormRoleRepository) FindPermissionByResourceAction(ctx context.Context, resource, action string) (*model.Permission, error) {
	var perm model.Permission
	err := r.db.WithContext(ctx).Where("resource = ? AND action = ?", resource, action).First(&perm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("permission not found")
		}
		return nil, err
	}
	return &perm, nil
}

// CountUsersWithRole returns the number of users assigned to a role
func (r *GormRoleRepository) CountUsersWithRole(ctx context.Context, roleID uint) (int64, error) {
	var count int64
	err := r.rbacScope.currentUserRoles(r.db.WithContext(ctx).Model(&model.UserRole{})).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}

// AssignRoleToUser assigns a role to a user
func (r *GormRoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	var count int64
	if err := r.rbacScope.currentRoles(r.db.WithContext(ctx).Model(&model.RBACRole{})).Where("id = ?", roleID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("role not found")
	}
	ur := model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).Create(&ur).Error
}

// RemoveRoleFromUser removes a role from a user
func (r *GormRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	result := r.rbacScope.currentUserRoles(r.db.WithContext(ctx)).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user role assignment not found")
	}
	return nil
}

// GetUserRoles returns all roles assigned to a user (with permissions preloaded)
func (r *GormRoleRepository) GetUserRoles(ctx context.Context, userID uint) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.rbacScope.currentUserRoles(r.db.WithContext(ctx)).
		Preload("Role", r.rbacScope.currentRoles).
		Preload("Role.Permissions", "resource <> ?", model.LegacyResourceSites).
		Where("user_id = ?", userID).
		Find(&userRoles).Error
	if err != nil {
		return nil, err
	}
	return userRoles, nil
}
