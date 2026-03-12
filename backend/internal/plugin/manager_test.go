package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestManager(t *testing.T) (*Manager, *Store, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	store := NewStore(db)
	require.NoError(t, store.AutoMigrate())

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "plugins")
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	require.NoError(t, os.MkdirAll(dataDir, 0o755))

	registry := provider.NewRegistry()
	mgr := NewManager(ManagerConfig{
		PluginDir: pluginDir,
		DataDir:   dataDir,
	}, store, registry)

	return mgr, store, pluginDir
}

func createPluginDir(t *testing.T, baseDir, pluginID string) string {
	t.Helper()
	dir := filepath.Join(baseDir, pluginID)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	manifest := `id: ` + pluginID + `
name: Test Plugin
version: 1.0.0
permissions:
  - network:outbound
providers:
  - type: storage
    name: ` + pluginID + `-storage
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644))
	return dir
}

func TestDiscoverPlugins(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	// Empty directory
	discovered, err := mgr.DiscoverPlugins(ctx)
	require.NoError(t, err)
	assert.Empty(t, discovered)

	// Create plugin directories
	createPluginDir(t, pluginDir, "test-plugin")
	createPluginDir(t, pluginDir, "another-plugin")

	// Add a non-plugin directory
	require.NoError(t, os.MkdirAll(filepath.Join(pluginDir, "not-a-plugin"), 0o755))

	// Add a regular file (should be skipped)
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "file.txt"), []byte("hello"), 0o644))

	discovered, err = mgr.DiscoverPlugins(ctx)
	require.NoError(t, err)
	assert.Len(t, discovered, 2)

	ids := make(map[string]bool)
	for _, d := range discovered {
		ids[d.ID] = true
	}
	assert.True(t, ids["test-plugin"])
	assert.True(t, ids["another-plugin"])
}

func TestDiscoverPlugins_NoDirectory(t *testing.T) {
	registry := provider.NewRegistry()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	store := NewStore(db)
	require.NoError(t, store.AutoMigrate())

	mgr := NewManager(ManagerConfig{
		PluginDir: "/nonexistent/path",
	}, store, registry)

	discovered, err := mgr.DiscoverPlugins(context.Background())
	require.NoError(t, err)
	assert.Empty(t, discovered)
}

func TestInstallPlugin(t *testing.T) {
	mgr, store, pluginDir := setupTestManager(t)
	ctx := context.Background()

	dir := createPluginDir(t, pluginDir, "test-plugin")

	meta, err := mgr.InstallPlugin(ctx, dir)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", meta.ID)
	assert.Equal(t, "1.0.0", meta.Version)

	// Verify DB record
	p, err := store.GetByPluginID(ctx, "test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "installed", p.State)
	assert.Equal(t, "Test Plugin", p.Name)
}

func TestInstallPlugin_AlreadyInstalled(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	dir := createPluginDir(t, pluginDir, "test-plugin")

	_, err := mgr.InstallPlugin(ctx, dir)
	require.NoError(t, err)

	_, err = mgr.InstallPlugin(ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already installed")
}

func TestInstallPlugin_InvalidManifest(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	// Create directory with invalid manifest
	dir := filepath.Join(pluginDir, "bad-plugin")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte("invalid: yaml: ["), 0o644))

	_, err := mgr.InstallPlugin(ctx, dir)
	assert.Error(t, err)
}

func TestInstallPlugin_InsufficientPermissions(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	// Create plugin with routes but no route:register permission
	dir := filepath.Join(pluginDir, "route-plugin")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	manifest := `id: route-plugin
name: Route Plugin
version: 1.0.0
permissions: []
routes:
  - method: GET
    path: /test
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644))

	_, err := mgr.InstallPlugin(ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}

func TestListPlugins(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	createPluginDir(t, pluginDir, "plugin-aaa")
	createPluginDir(t, pluginDir, "plugin-bbb")

	_, err := mgr.InstallPlugin(ctx, filepath.Join(pluginDir, "plugin-aaa"))
	require.NoError(t, err)
	_, err = mgr.InstallPlugin(ctx, filepath.Join(pluginDir, "plugin-bbb"))
	require.NoError(t, err)

	plugins, err := mgr.ListPlugins(ctx)
	require.NoError(t, err)
	assert.Len(t, plugins, 2)
}

func TestGetPlugin(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	createPluginDir(t, pluginDir, "test-plugin")
	_, err := mgr.InstallPlugin(ctx, filepath.Join(pluginDir, "test-plugin"))
	require.NoError(t, err)

	p, err := mgr.GetPlugin(ctx, "test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", p.PluginID)
}

func TestGetPlugin_NotFound(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	_, err := mgr.GetPlugin(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestUninstallPlugin(t *testing.T) {
	mgr, store, pluginDir := setupTestManager(t)
	ctx := context.Background()

	createPluginDir(t, pluginDir, "test-plugin")
	_, err := mgr.InstallPlugin(ctx, filepath.Join(pluginDir, "test-plugin"))
	require.NoError(t, err)

	err = mgr.UninstallPlugin(ctx, "test-plugin")
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "test-plugin")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestStopAll(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	// No hosts running - should be a no-op
	err := mgr.StopAll()
	assert.NoError(t, err)
}

func TestManagerConfig_Defaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	store := NewStore(db)
	registry := provider.NewRegistry()

	mgr := NewManager(ManagerConfig{}, store, registry)
	assert.Equal(t, "./plugins", mgr.config.PluginDir)
	assert.Equal(t, "./data/plugins", mgr.config.DataDir)
}

func TestUpdateSettings(t *testing.T) {
	mgr, store, pluginDir := setupTestManager(t)
	ctx := context.Background()

	createPluginDir(t, pluginDir, "test-plugin")
	_, err := mgr.InstallPlugin(ctx, filepath.Join(pluginDir, "test-plugin"))
	require.NoError(t, err)

	settings := map[string]any{
		"endpoint": "https://s3.example.com",
		"bucket":   "my-bucket",
	}

	err = mgr.UpdateSettings(ctx, "test-plugin", settings)
	require.NoError(t, err)

	p, err := store.GetByPluginID(ctx, "test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "https://s3.example.com", p.Settings["endpoint"])
}

func TestStartEnabledPlugins_NoEnabled(t *testing.T) {
	mgr, _, pluginDir := setupTestManager(t)
	ctx := context.Background()

	createPluginDir(t, pluginDir, "test-plugin")
	_, err := mgr.InstallPlugin(ctx, filepath.Join(pluginDir, "test-plugin"))
	require.NoError(t, err)

	// No enabled plugins - should be a no-op
	err = mgr.StartEnabledPlugins(ctx)
	assert.NoError(t, err)
}

func TestFindBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// No binary
	assert.Equal(t, "", findBinary(tmpDir, "my-plugin"))

	// Binary named after plugin ID
	binPath := filepath.Join(tmpDir, "my-plugin")
	require.NoError(t, os.WriteFile(binPath, []byte("binary"), 0o755))
	assert.Equal(t, binPath, findBinary(tmpDir, "my-plugin"))

	// Clean and try "plugin" name
	tmpDir2 := t.TempDir()
	pluginBin := filepath.Join(tmpDir2, "plugin")
	require.NoError(t, os.WriteFile(pluginBin, []byte("binary"), 0o755))
	assert.Equal(t, pluginBin, findBinary(tmpDir2, "other-id"))
}

func TestSettingsToStringMap(t *testing.T) {
	m := model.JSONMap{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	result := settingsToStringMap(m)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "42", result["key2"])
	assert.Equal(t, "true", result["key3"])
}
