package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandbox_Check(t *testing.T) {
	s := NewSandbox("test-plugin", []Permission{PermNetworkOutbound, PermDatabaseRead})

	assert.NoError(t, s.Check(PermNetworkOutbound))
	assert.NoError(t, s.Check(PermDatabaseRead))
	assert.Error(t, s.Check(PermDatabaseWrite))
	assert.Error(t, s.Check(PermFileSystemRead))
}

func TestSandbox_CheckAll(t *testing.T) {
	s := NewSandbox("test-plugin", []Permission{PermNetworkOutbound, PermDatabaseRead})

	assert.NoError(t, s.CheckAll(PermNetworkOutbound))
	assert.NoError(t, s.CheckAll(PermNetworkOutbound, PermDatabaseRead))
	assert.Error(t, s.CheckAll(PermNetworkOutbound, PermDatabaseWrite))
}

func TestSandbox_CheckAll_MultipleMissing(t *testing.T) {
	s := NewSandbox("test-plugin", []Permission{})

	err := s.CheckAll(PermDatabaseRead, PermDatabaseWrite)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database:read")
	assert.Contains(t, err.Error(), "database:write")
}

func TestSandbox_HasPermission(t *testing.T) {
	s := NewSandbox("test-plugin", []Permission{PermNetworkOutbound})

	assert.True(t, s.HasPermission(PermNetworkOutbound))
	assert.False(t, s.HasPermission(PermDatabaseRead))
}

func TestSandbox_EmptyPermissions(t *testing.T) {
	s := NewSandbox("test-plugin", nil)

	assert.Error(t, s.Check(PermNetworkOutbound))
	assert.False(t, s.HasPermission(PermNetworkOutbound))
}

func TestRequiredPermissionsForProvider(t *testing.T) {
	tests := []struct {
		providerType string
		expected     []Permission
	}{
		{"storage", []Permission{PermNetworkOutbound}},
		{"search", []Permission{PermNetworkOutbound}},
		{"notifier", []Permission{PermNetworkOutbound}},
		{"captcha", []Permission{PermNetworkOutbound}},
		{"unknown", nil},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			result := RequiredPermissionsForProvider(tt.providerType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateManifestPermissions_AllProviders(t *testing.T) {
	meta := &PluginMeta{
		ID:      "full-plugin",
		Name:    "Full Plugin",
		Version: "1.0.0",
		Permissions: []Permission{
			PermNetworkOutbound,
			PermRouteRegister,
			PermFrontendInject,
		},
		Providers: []ProviderDecl{
			{Type: "storage", Name: "s3"},
			{Type: "notifier", Name: "email"},
		},
		Routes: []RouteDecl{
			{Method: "GET", Path: "/test"},
		},
		FrontendEntry: "frontend/dist/index.js",
	}

	assert.NoError(t, ValidateManifestPermissions(meta))
}

func TestValidateManifestPermissions_MissingProviderPerm(t *testing.T) {
	meta := &PluginMeta{
		ID:          "storage-plugin",
		Name:        "Storage Plugin",
		Version:     "1.0.0",
		Permissions: []Permission{}, // missing network:outbound
		Providers: []ProviderDecl{
			{Type: "storage", Name: "s3"},
		},
	}

	err := ValidateManifestPermissions(meta)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network:outbound")
}

func TestValidateManifestPermissions_MissingRoutePerm(t *testing.T) {
	meta := &PluginMeta{
		ID:          "route-plugin",
		Name:        "Route Plugin",
		Version:     "1.0.0",
		Permissions: []Permission{}, // missing route:register
		Routes: []RouteDecl{
			{Method: "GET", Path: "/test"},
		},
	}

	err := ValidateManifestPermissions(meta)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route:register")
}

func TestValidateManifestPermissions_MissingFrontendPerm(t *testing.T) {
	meta := &PluginMeta{
		ID:            "frontend-plugin",
		Name:          "Frontend Plugin",
		Version:       "1.0.0",
		Permissions:   []Permission{}, // missing frontend:inject
		FrontendEntry: "frontend/dist/index.js",
	}

	err := ValidateManifestPermissions(meta)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frontend:inject")
}

func TestValidateManifestPermissions_NoRequirements(t *testing.T) {
	meta := &PluginMeta{
		ID:      "simple-plugin",
		Name:    "Simple Plugin",
		Version: "1.0.0",
	}

	assert.NoError(t, ValidateManifestPermissions(meta))
}
