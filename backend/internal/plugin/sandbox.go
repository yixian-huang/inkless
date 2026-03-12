package plugin

import (
	"fmt"
	"strings"
)

// Sandbox enforces permissions declared in a plugin's manifest.
// Before a plugin is allowed to perform an operation, the sandbox checks
// that the required permission was declared in plugin.yaml and approved.
type Sandbox struct {
	pluginID    string
	permissions map[Permission]bool
}

// NewSandbox creates a sandbox for a plugin with the given permissions.
func NewSandbox(pluginID string, perms []Permission) *Sandbox {
	m := make(map[Permission]bool, len(perms))
	for _, p := range perms {
		m[p] = true
	}
	return &Sandbox{
		pluginID:    pluginID,
		permissions: m,
	}
}

// Check returns nil if the plugin has the requested permission,
// or an error describing the denied capability.
func (s *Sandbox) Check(perm Permission) error {
	if s.permissions[perm] {
		return nil
	}
	return fmt.Errorf("plugin %q does not have permission %q", s.pluginID, perm)
}

// CheckAll returns nil if all given permissions are granted.
func (s *Sandbox) CheckAll(perms ...Permission) error {
	var denied []string
	for _, p := range perms {
		if !s.permissions[p] {
			denied = append(denied, string(p))
		}
	}
	if len(denied) > 0 {
		return fmt.Errorf("plugin %q missing permissions: %s", s.pluginID, strings.Join(denied, ", "))
	}
	return nil
}

// HasPermission returns true if the plugin has the given permission.
func (s *Sandbox) HasPermission(perm Permission) bool {
	return s.permissions[perm]
}

// RequiredPermissionsForProvider returns the minimum permissions
// needed to implement a given provider type.
func RequiredPermissionsForProvider(providerType string) []Permission {
	switch providerType {
	case "storage":
		return []Permission{PermNetworkOutbound}
	case "search":
		return []Permission{PermNetworkOutbound}
	case "notifier":
		return []Permission{PermNetworkOutbound}
	case "captcha":
		return []Permission{PermNetworkOutbound}
	default:
		return nil
	}
}

// ValidateManifestPermissions checks that a plugin's declared permissions
// are sufficient for its declared providers and routes.
func ValidateManifestPermissions(meta *PluginMeta) error {
	sandbox := NewSandbox(meta.ID, meta.Permissions)
	var errs []string

	// Check provider permissions
	for _, prov := range meta.Providers {
		required := RequiredPermissionsForProvider(prov.Type)
		for _, perm := range required {
			if err := sandbox.Check(perm); err != nil {
				errs = append(errs, fmt.Sprintf("provider %q (%s) requires %s", prov.Name, prov.Type, perm))
			}
		}
	}

	// Check route permissions
	if len(meta.Routes) > 0 {
		if err := sandbox.Check(PermRouteRegister); err != nil {
			errs = append(errs, fmt.Sprintf("routes declared but missing %s permission", PermRouteRegister))
		}
	}

	// Check frontend injection permissions
	if meta.FrontendEntry != "" {
		if err := sandbox.Check(PermFrontendInject); err != nil {
			errs = append(errs, fmt.Sprintf("frontendEntry declared but missing %s permission", PermFrontendInject))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("permission validation failed for plugin %q: %s", meta.ID, strings.Join(errs, "; "))
	}
	return nil
}
