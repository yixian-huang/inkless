package repository

import (
	"context"

	"blotting-consultancy/internal/model"
)

// RoleRepository defines the interface for RBAC role data access
type RoleRepository interface {
	// Create creates a new role
	Create(ctx context.Context, role *model.RBACRole) error

	// FindByID finds a role by ID, including its permissions
	FindByID(ctx context.Context, id uint) (*model.RBACRole, error)

	// FindByName finds a role by name, including its permissions
	FindByName(ctx context.Context, name string) (*model.RBACRole, error)

	// Update updates an existing role
	Update(ctx context.Context, role *model.RBACRole) error

	// Delete deletes a role by ID
	Delete(ctx context.Context, id uint) error

	// List returns all roles with their permissions
	List(ctx context.Context) ([]*model.RBACRole, error)

	// SetPermissions replaces all permissions for a role
	SetPermissions(ctx context.Context, roleID uint, permissionIDs []uint) error

	// ListPermissions returns all available permissions
	ListPermissions(ctx context.Context) ([]*model.Permission, error)

	// CreatePermission creates a new permission
	CreatePermission(ctx context.Context, perm *model.Permission) error

	// FindPermissionByResourceAction finds a permission by resource and action
	FindPermissionByResourceAction(ctx context.Context, resource, action string) (*model.Permission, error)

	// CountUsersWithRole returns the number of users assigned to a role
	CountUsersWithRole(ctx context.Context, roleID uint) (int64, error)

	// AssignRoleToUser assigns a role to a user
	AssignRoleToUser(ctx context.Context, userID, roleID uint) error

	// RemoveRoleFromUser removes a role from a user
	RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error

	// GetUserRoles returns all roles assigned to a user (with permissions preloaded)
	GetUserRoles(ctx context.Context, userID uint) ([]model.UserRole, error)
}
