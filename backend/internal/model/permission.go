package model

import "fmt"

// Permission represents a granular permission in the RBAC system.
// Permissions follow the "resource:action" pattern.
type Permission struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Resource    string `gorm:"not null;size:50;uniqueIndex:idx_resource_action" json:"resource"`
	Action      string `gorm:"not null;size:50;uniqueIndex:idx_resource_action" json:"action"`
	Description string `gorm:"type:text" json:"description"`
}

// TableName overrides the default table name
func (Permission) TableName() string {
	return "permissions"
}

// Key returns the permission key in "resource:action" format
func (p Permission) Key() string {
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

// MatchesPermission checks if this permission matches the given resource:action pair.
// Supports wildcard matching: "*:*" matches everything, "articles:*" matches all article actions.
func (p Permission) Matches(resource, action string) bool {
	resourceMatch := p.Resource == "*" || p.Resource == resource
	actionMatch := p.Action == "*" || p.Action == action
	return resourceMatch && actionMatch
}

// BuiltinResources lists all resource types in the system
var BuiltinResources = []string{
	"dashboard", "articles", "pages", "media", "comments",
	"categories", "tags", "menus", "themes", "analytics",
	"audit_logs", "backups", "users", "form_submissions",
	"roles", "settings", "system", "plugins",
}

// LegacyResourceSites identifies permission rows left by the retired
// experimental shared-database multi-site feature. Rows remain stored for the
// rollback window but must not be exposed through current role APIs.
const LegacyResourceSites = "sites"

// BuiltinActions lists all action types in the system
var BuiltinActions = []string{
	"create", "read", "update", "delete", "publish", "manage",
}
