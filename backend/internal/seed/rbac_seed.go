package seed

import (
	"context"
	"fmt"
	"log"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// RBACSeedConfig defines the built-in roles and their permission sets
type RBACSeedConfig struct {
	Name        string
	DisplayName string
	Description string
	// Permissions as "resource:action" pairs. "*" is a wildcard.
	Permissions []string
}

// BuiltinRoles defines the default RBAC roles and their permissions
var BuiltinRoles = []RBACSeedConfig{
	{
		Name:        model.BuiltinRoleSuperAdmin,
		DisplayName: "Super Admin",
		Description: "Full system access including system-level operations",
		Permissions: []string{"*:*"},
	},
	{
		Name:        model.BuiltinRoleSiteAdmin,
		DisplayName: "Site Admin",
		Description: "Full site management access except system-level operations",
		Permissions: func() []string {
			var perms []string
			for _, res := range model.BuiltinResources {
				if res == "system" {
					continue
				}
				for _, act := range model.BuiltinActions {
					if res == "users" && act == "delete" {
						continue
					}
					perms = append(perms, fmt.Sprintf("%s:%s", res, act))
				}
			}
			return perms
		}(),
	},
	{
		Name:        model.BuiltinRoleEditor,
		DisplayName: "Editor",
		Description: "Content editing access for articles, pages, media, and related resources",
		Permissions: []string{
			"articles:create", "articles:read", "articles:update", "articles:delete",
			"pages:create", "pages:read", "pages:update", "pages:delete",
			"media:create", "media:read", "media:update", "media:delete",
			"comments:create", "comments:read", "comments:update", "comments:delete",
			"categories:create", "categories:read", "categories:update", "categories:delete",
			"tags:create", "tags:read", "tags:update", "tags:delete",
			"menus:read",
			"dashboard:read",
		},
	},
	{
		Name:        model.BuiltinRoleAuthor,
		DisplayName: "Author",
		Description: "Content creation access with limited editing to own content",
		Permissions: []string{
			"articles:create", "articles:read", "articles:update",
			"media:create", "media:read",
			"comments:read",
			"dashboard:read",
		},
	},
	{
		Name:        model.BuiltinRoleViewer,
		DisplayName: "Viewer",
		Description: "Read-only access to all resources",
		Permissions: func() []string {
			var perms []string
			for _, res := range model.BuiltinResources {
				perms = append(perms, fmt.Sprintf("%s:read", res))
			}
			return perms
		}(),
	},
}

// SeedRBAC seeds built-in roles and permissions. It is idempotent.
func SeedRBAC(ctx context.Context, roleRepo repository.RoleRepository) error {
	log.Println("Seeding RBAC permissions and roles...")

	// 1. Seed all permissions (resource:action combinations)
	permMap, err := seedPermissions(ctx, roleRepo)
	if err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}

	// 2. Seed built-in roles with their permission sets
	for _, roleCfg := range BuiltinRoles {
		if err := seedRole(ctx, roleRepo, roleCfg, permMap); err != nil {
			return fmt.Errorf("seed role %s: %w", roleCfg.Name, err)
		}
	}

	log.Println("RBAC seed completed successfully")
	return nil
}

// seedPermissions creates all resource:action permission combinations
func seedPermissions(ctx context.Context, roleRepo repository.RoleRepository) (map[string]uint, error) {
	permMap := make(map[string]uint)

	// Create the wildcard permission
	wildcardPerms := []struct{ Resource, Action, Desc string }{
		{"*", "*", "Full access to all resources and actions"},
	}

	// Add all resource:action combinations
	for _, res := range model.BuiltinResources {
		for _, act := range model.BuiltinActions {
			desc := fmt.Sprintf("%s %s access", act, res)
			wildcardPerms = append(wildcardPerms, struct{ Resource, Action, Desc string }{
				res, act, desc,
			})
		}
	}

	for _, p := range wildcardPerms {
		existing, err := roleRepo.FindPermissionByResourceAction(ctx, p.Resource, p.Action)
		if err == nil && existing != nil {
			permMap[existing.Key()] = existing.ID
			continue
		}

		perm := &model.Permission{
			Resource:    p.Resource,
			Action:      p.Action,
			Description: p.Desc,
		}
		if err := roleRepo.CreatePermission(ctx, perm); err != nil {
			return nil, fmt.Errorf("create permission %s:%s: %w", p.Resource, p.Action, err)
		}
		permMap[perm.Key()] = perm.ID
		log.Printf("Created permission: %s:%s", p.Resource, p.Action)
	}

	return permMap, nil
}

// seedRole creates a built-in role if it doesn't exist and assigns permissions
func seedRole(ctx context.Context, roleRepo repository.RoleRepository, cfg RBACSeedConfig, permMap map[string]uint) error {
	existing, err := roleRepo.FindByName(ctx, cfg.Name)
	role := existing
	if err != nil || role == nil {
		role = &model.RBACRole{
			Name:        cfg.Name,
			DisplayName: cfg.DisplayName,
			Description: cfg.Description,
			IsSystem:    true,
		}

		if err := roleRepo.Create(ctx, role); err != nil {
			return fmt.Errorf("create role: %w", err)
		}
		log.Printf("Created role: %s (id=%d)", cfg.Name, role.ID)
	} else {
		log.Printf("Synchronizing permissions for role %s (id=%d)", cfg.Name, role.ID)
	}

	// Collect permission IDs for this role
	var permIDs []uint
	for _, permKey := range cfg.Permissions {
		id, ok := permMap[permKey]
		if !ok {
			return fmt.Errorf("permission %s not found in map", permKey)
		}
		permIDs = append(permIDs, id)
	}

	if len(permIDs) > 0 {
		if err := roleRepo.SetPermissions(ctx, role.ID, permIDs); err != nil {
			return fmt.Errorf("set permissions: %w", err)
		}
		log.Printf("Assigned %d permissions to role %s", len(permIDs), cfg.Name)
	}

	return nil
}
