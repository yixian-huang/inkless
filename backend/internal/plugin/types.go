package plugin

import (
	"fmt"
	"regexp"
	"strings"
)

// PluginState represents the lifecycle state of a plugin.
type PluginState string

const (
	StateInstalled PluginState = "installed"
	StateEnabled   PluginState = "enabled"
	StateDisabled  PluginState = "disabled"
	StateFailed    PluginState = "failed"
)

// ValidStates returns all valid plugin states.
func ValidStates() []PluginState {
	return []PluginState{StateInstalled, StateEnabled, StateDisabled, StateFailed}
}

// IsValidState checks if a given state is valid.
func IsValidState(s PluginState) bool {
	for _, v := range ValidStates() {
		if v == s {
			return true
		}
	}
	return false
}

// Permission represents a capability a plugin requests.
type Permission string

const (
	PermDatabaseRead    Permission = "database:read"
	PermDatabaseWrite   Permission = "database:write"
	PermFileSystemRead  Permission = "filesystem:read"
	PermFileSystemWrite Permission = "filesystem:write"
	PermNetworkOutbound Permission = "network:outbound"
	PermEventSubscribe  Permission = "event:subscribe"
	PermEventPublish    Permission = "event:publish"
	PermHookRegister    Permission = "hook:register"
	PermRouteRegister   Permission = "route:register"
	PermFrontendInject  Permission = "frontend:inject"
)

// AllPermissions returns the full list of valid permissions.
func AllPermissions() []Permission {
	return []Permission{
		PermDatabaseRead, PermDatabaseWrite,
		PermFileSystemRead, PermFileSystemWrite,
		PermNetworkOutbound,
		PermEventSubscribe, PermEventPublish,
		PermHookRegister, PermRouteRegister,
		PermFrontendInject,
	}
}

// IsValidPermission checks if a permission string is valid.
func IsValidPermission(p Permission) bool {
	for _, v := range AllPermissions() {
		if v == p {
			return true
		}
	}
	return false
}

// PluginMeta represents the parsed content of plugin.yaml.
type PluginMeta struct {
	ID             string         `yaml:"id" json:"id"`
	Name           string         `yaml:"name" json:"name"`
	NameZh         string         `yaml:"nameZh" json:"nameZh"`
	Version        string         `yaml:"version" json:"version"`
	Description    string         `yaml:"description" json:"description"`
	Author         string         `yaml:"author" json:"author"`
	License        string         `yaml:"license" json:"license"`
	Homepage       string         `yaml:"homepage" json:"homepage"`
	MinAppVersion  string         `yaml:"minAppVersion" json:"minAppVersion"`
	Dependencies   []Dependency   `yaml:"dependencies" json:"dependencies"`
	Permissions    []Permission   `yaml:"permissions" json:"permissions"`
	Providers      []ProviderDecl `yaml:"providers" json:"providers"`
	Routes         []RouteDecl    `yaml:"routes" json:"routes"`
	FrontendEntry  string         `yaml:"frontendEntry" json:"frontendEntry"`
	SettingsSchema map[string]any `yaml:"settingsSchema" json:"settingsSchema"`
}

// Dependency declares a required plugin.
type Dependency struct {
	PluginID   string `yaml:"pluginId" json:"pluginId"`
	MinVersion string `yaml:"minVersion" json:"minVersion"`
}

// ProviderDecl declares which provider interface a plugin implements.
type ProviderDecl struct {
	Type string `yaml:"type" json:"type"` // "storage", "search", "notifier", "captcha"
	Name string `yaml:"name" json:"name"` // registration key in Provider Registry
}

// RouteDecl declares an API route the plugin wants to register.
type RouteDecl struct {
	Method string `yaml:"method" json:"method"`
	Path   string `yaml:"path" json:"path"`
}

// ValidProviderTypes returns the list of valid provider type strings.
func ValidProviderTypes() []string {
	return []string{"storage", "search", "notifier", "captcha"}
}

// pluginIDPattern validates plugin IDs: lowercase letter followed by 2-49 lowercase alphanumeric or hyphens.
var pluginIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,49}$`)

// semverPattern validates semantic versioning strings (e.g., "1.0.0", "2.1.3-beta.1").
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Validate checks that all required fields are present and valid.
func (m *PluginMeta) Validate() error {
	var errs []string

	if m.ID == "" {
		errs = append(errs, "id is required")
	} else if !pluginIDPattern.MatchString(m.ID) {
		errs = append(errs, fmt.Sprintf("id %q must match pattern ^[a-z][a-z0-9-]{2,49}$", m.ID))
	}

	if m.Name == "" {
		errs = append(errs, "name is required")
	}

	if m.Version == "" {
		errs = append(errs, "version is required")
	} else if !semverPattern.MatchString(m.Version) {
		errs = append(errs, fmt.Sprintf("version %q is not valid semver", m.Version))
	}

	// Validate permissions
	for _, p := range m.Permissions {
		if !IsValidPermission(p) {
			errs = append(errs, fmt.Sprintf("unknown permission %q", p))
		}
	}

	// Validate provider types
	validTypes := ValidProviderTypes()
	for _, prov := range m.Providers {
		found := false
		for _, vt := range validTypes {
			if prov.Type == vt {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("unknown provider type %q", prov.Type))
		}
	}

	// Validate minAppVersion if provided
	if m.MinAppVersion != "" && !semverPattern.MatchString(m.MinAppVersion) {
		errs = append(errs, fmt.Sprintf("minAppVersion %q is not valid semver", m.MinAppVersion))
	}

	// Validate dependency versions if provided
	for _, dep := range m.Dependencies {
		if dep.PluginID == "" {
			errs = append(errs, "dependency pluginId is required")
		}
		if dep.MinVersion != "" && !semverPattern.MatchString(dep.MinVersion) {
			errs = append(errs, fmt.Sprintf("dependency %q minVersion %q is not valid semver", dep.PluginID, dep.MinVersion))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("manifest validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
