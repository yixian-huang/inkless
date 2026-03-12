package plugin

import (
	"context"
	"testing"

	"blotting-consultancy/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Plugin{}, &model.PluginSetting{}))
	return db
}

func newTestPlugin(id string) *model.Plugin {
	return &model.Plugin{
		PluginID:    id,
		Name:        "Test Plugin " + id,
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		State:       string(StateInstalled),
		Source:      "local",
		Permissions: model.JSONStringSlice{"database:read"},
	}
}

func TestStore_Create_And_GetByPluginID(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	p := newTestPlugin("my-plugin")
	require.NoError(t, store.Create(ctx, p))
	assert.NotZero(t, p.ID)

	got, err := store.GetByPluginID(ctx, "my-plugin")
	require.NoError(t, err)
	assert.Equal(t, "my-plugin", got.PluginID)
	assert.Equal(t, "Test Plugin my-plugin", got.Name)
	assert.Equal(t, "1.0.0", got.Version)
	assert.Equal(t, string(StateInstalled), got.State)
	assert.Equal(t, "local", got.Source)
	assert.Equal(t, model.JSONStringSlice{"database:read"}, got.Permissions)
}

func TestStore_GetByPluginID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	_, err := store.GetByPluginID(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Create_DuplicateID(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	p1 := newTestPlugin("dup-plugin")
	require.NoError(t, store.Create(ctx, p1))

	p2 := newTestPlugin("dup-plugin")
	err := store.Create(ctx, p2)
	require.Error(t, err) // unique constraint violation
}

func TestStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, newTestPlugin("plugin-a")))
	require.NoError(t, store.Create(ctx, newTestPlugin("plugin-b")))
	require.NoError(t, store.Create(ctx, newTestPlugin("plugin-c")))

	plugins, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, plugins, 3)
}

func TestStore_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	plugins, err := store.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestStore_ListByState(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	p1 := newTestPlugin("plugin-a")
	p1.State = string(StateEnabled)
	require.NoError(t, store.Create(ctx, p1))

	p2 := newTestPlugin("plugin-b")
	p2.State = string(StateDisabled)
	require.NoError(t, store.Create(ctx, p2))

	p3 := newTestPlugin("plugin-c")
	p3.State = string(StateEnabled)
	require.NoError(t, store.Create(ctx, p3))

	enabled, err := store.ListByState(ctx, StateEnabled)
	require.NoError(t, err)
	assert.Len(t, enabled, 2)

	disabled, err := store.ListByState(ctx, StateDisabled)
	require.NoError(t, err)
	assert.Len(t, disabled, 1)
}

func TestStore_UpdateState(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, newTestPlugin("my-plugin")))

	// Update to enabled
	require.NoError(t, store.UpdateState(ctx, "my-plugin", StateEnabled, ""))

	got, err := store.GetByPluginID(ctx, "my-plugin")
	require.NoError(t, err)
	assert.Equal(t, string(StateEnabled), got.State)
	assert.Empty(t, got.ErrorMsg)

	// Update to failed with error message
	require.NoError(t, store.UpdateState(ctx, "my-plugin", StateFailed, "crashed on startup"))

	got, err = store.GetByPluginID(ctx, "my-plugin")
	require.NoError(t, err)
	assert.Equal(t, string(StateFailed), got.State)
	assert.Equal(t, "crashed on startup", got.ErrorMsg)
}

func TestStore_UpdateState_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	err := store.UpdateState(ctx, "nonexistent", StateEnabled, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_UpdateSettings(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, newTestPlugin("my-plugin")))

	settings := map[string]any{
		"endpoint": "https://s3.amazonaws.com",
		"bucket":   "my-bucket",
		"port":     float64(443),
	}
	require.NoError(t, store.UpdateSettings(ctx, "my-plugin", settings))

	got, err := store.GetByPluginID(ctx, "my-plugin")
	require.NoError(t, err)
	assert.Equal(t, "https://s3.amazonaws.com", got.Settings["endpoint"])
	assert.Equal(t, "my-bucket", got.Settings["bucket"])
}

func TestStore_UpdateSettings_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	err := store.UpdateSettings(ctx, "nonexistent", map[string]any{"key": "val"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, newTestPlugin("my-plugin")))

	require.NoError(t, store.Delete(ctx, "my-plugin"))

	_, err := store.GetByPluginID(ctx, "my-plugin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Verify list is empty too
	plugins, err := store.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestStore_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Exists(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	exists, err := store.Exists(ctx, "my-plugin")
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, store.Create(ctx, newTestPlugin("my-plugin")))

	exists, err = store.Exists(ctx, "my-plugin")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestStore_AutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)

	store := NewStore(db)
	require.NoError(t, store.AutoMigrate())

	// Verify we can create a plugin after migration
	ctx := context.Background()
	require.NoError(t, store.Create(ctx, newTestPlugin("test-plugin")))

	got, err := store.GetByPluginID(ctx, "test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", got.PluginID)
}
