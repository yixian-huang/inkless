package model

import (
	"errors"
	"regexp"
	"time"
)

// RBACRole represents a role in the RBAC system.
// Named RBACRole to avoid conflict with the legacy Role type alias.
type RBACRole struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Name        string       `gorm:"uniqueIndex;not null;size:50" json:"name"`
	DisplayName string       `gorm:"not null;size:100" json:"displayName"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"not null;default:false" json:"isSystem"`
	SiteID      *uint        `gorm:"index" json:"siteId,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
	CreatedAt   time.Time    `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time    `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName overrides the default table name
func (RBACRole) TableName() string {
	return "roles"
}

// roleNameRegex validates role names: lowercase alphanumeric + underscores + hyphens
var roleNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,48}[a-z0-9]$`)

// Validate validates the role model
func (r *RBACRole) Validate() error {
	if r.Name == "" {
		return errors.New("role name is required")
	}
	if len(r.Name) == 1 {
		if !regexp.MustCompile(`^[a-z]$`).MatchString(r.Name) {
			return errors.New("role name must contain only lowercase alphanumeric characters, hyphens, and underscores")
		}
	} else if !roleNameRegex.MatchString(r.Name) {
		return errors.New("role name must contain only lowercase alphanumeric characters, hyphens, and underscores")
	}
	if r.DisplayName == "" {
		return errors.New("role display name is required")
	}
	return nil
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID uint  `gorm:"primaryKey" json:"userId"`
	RoleID uint  `gorm:"primaryKey" json:"roleId"`
	SiteID *uint `json:"siteId,omitempty"` // NULL = global assignment (Phase 4.2)
	Role   RBACRole `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// TableName overrides the default table name
func (UserRole) TableName() string {
	return "user_roles"
}

// BuiltinRoleSuperAdmin is the super admin role name
const BuiltinRoleSuperAdmin = "super_admin"

// BuiltinRoleSiteAdmin is the site admin role name
const BuiltinRoleSiteAdmin = "site_admin"

// BuiltinRoleEditor is the editor role name
const BuiltinRoleEditor = "editor"

// BuiltinRoleAuthor is the author role name
const BuiltinRoleAuthor = "author"

// BuiltinRoleViewer is the viewer role name
const BuiltinRoleViewer = "viewer"

// BuiltinRoleNames contains all built-in role names
var BuiltinRoleNames = []string{
	BuiltinRoleSuperAdmin,
	BuiltinRoleSiteAdmin,
	BuiltinRoleEditor,
	BuiltinRoleAuthor,
	BuiltinRoleViewer,
}
