package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllPermissions(t *testing.T) {
	perms := AllPermissions()
	assert.Len(t, perms, 10, "should have 10 permissions")

	expected := []Permission{
		PermDatabaseRead, PermDatabaseWrite,
		PermFileSystemRead, PermFileSystemWrite,
		PermNetworkOutbound,
		PermEventSubscribe, PermEventPublish,
		PermHookRegister, PermRouteRegister,
		PermFrontendInject,
	}
	assert.Equal(t, expected, perms)
}

func TestIsValidPermission(t *testing.T) {
	assert.True(t, IsValidPermission(PermDatabaseRead))
	assert.True(t, IsValidPermission(PermNetworkOutbound))
	assert.False(t, IsValidPermission(Permission("invalid:perm")))
	assert.False(t, IsValidPermission(Permission("")))
}

func TestValidStates(t *testing.T) {
	states := ValidStates()
	assert.Len(t, states, 4)
	assert.Contains(t, states, StateInstalled)
	assert.Contains(t, states, StateEnabled)
	assert.Contains(t, states, StateDisabled)
	assert.Contains(t, states, StateFailed)
}

func TestIsValidState(t *testing.T) {
	assert.True(t, IsValidState(StateInstalled))
	assert.True(t, IsValidState(StateEnabled))
	assert.True(t, IsValidState(StateDisabled))
	assert.True(t, IsValidState(StateFailed))
	assert.False(t, IsValidState(PluginState("unknown")))
	assert.False(t, IsValidState(PluginState("")))
}

func TestPluginMetaValidate(t *testing.T) {
	tests := []struct {
		name    string
		meta    PluginMeta
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal meta",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid full meta",
			meta: PluginMeta{
				ID:            "s3-storage",
				Name:          "S3 Storage",
				NameZh:        "S3 对象存储",
				Version:       "1.2.3",
				Description:   "Store files in S3",
				Author:        "Test",
				License:       "MIT",
				Homepage:      "https://example.com",
				MinAppVersion: "1.0.0",
				Dependencies: []Dependency{
					{PluginID: "base-plugin", MinVersion: "0.1.0"},
				},
				Permissions: []Permission{PermNetworkOutbound, PermFileSystemRead},
				Providers: []ProviderDecl{
					{Type: "storage", Name: "s3-storage"},
				},
				Routes: []RouteDecl{
					{Method: "GET", Path: "/buckets"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			meta: PluginMeta{
				Name:    "My Plugin",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name: "invalid id format - uppercase",
			meta: PluginMeta{
				ID:      "MyPlugin",
				Name:    "My Plugin",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "must match pattern",
		},
		{
			name: "invalid id format - too short",
			meta: PluginMeta{
				ID:      "ab",
				Name:    "My Plugin",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "must match pattern",
		},
		{
			name: "invalid id format - starts with number",
			meta: PluginMeta{
				ID:      "1plugin",
				Name:    "My Plugin",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "must match pattern",
		},
		{
			name: "missing name",
			meta: PluginMeta{
				ID:      "my-plugin",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing version",
			meta: PluginMeta{
				ID:   "my-plugin",
				Name: "My Plugin",
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "invalid version - not semver",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0",
			},
			wantErr: true,
			errMsg:  "not valid semver",
		},
		{
			name: "invalid permission",
			meta: PluginMeta{
				ID:          "my-plugin",
				Name:        "My Plugin",
				Version:     "1.0.0",
				Permissions: []Permission{"invalid:perm"},
			},
			wantErr: true,
			errMsg:  "unknown permission",
		},
		{
			name: "invalid provider type",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0.0",
				Providers: []ProviderDecl{
					{Type: "unknown", Name: "test"},
				},
			},
			wantErr: true,
			errMsg:  "unknown provider type",
		},
		{
			name: "invalid minAppVersion",
			meta: PluginMeta{
				ID:            "my-plugin",
				Name:          "My Plugin",
				Version:       "1.0.0",
				MinAppVersion: "bad",
			},
			wantErr: true,
			errMsg:  "not valid semver",
		},
		{
			name: "invalid dependency version",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{PluginID: "other", MinVersion: "nope"},
				},
			},
			wantErr: true,
			errMsg:  "not valid semver",
		},
		{
			name: "dependency missing pluginId",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{PluginID: "", MinVersion: "1.0.0"},
				},
			},
			wantErr: true,
			errMsg:  "dependency pluginId is required",
		},
		{
			name: "valid semver with pre-release",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0.0-beta.1",
			},
			wantErr: false,
		},
		{
			name: "valid semver with build metadata",
			meta: PluginMeta{
				ID:      "my-plugin",
				Name:    "My Plugin",
				Version: "1.0.0+build.123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidProviderTypes(t *testing.T) {
	types := ValidProviderTypes()
	assert.Len(t, types, 4)
	assert.Contains(t, types, "storage")
	assert.Contains(t, types, "search")
	assert.Contains(t, types, "notifier")
	assert.Contains(t, types, "captcha")
}
