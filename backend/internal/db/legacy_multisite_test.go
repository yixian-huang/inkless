package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/model"
)

func TestInspectLegacyMultiSiteSchemaCountsDataWithoutMutation(t *testing.T) {
	database, err := Init(InitOptions{DSN: ":memory:", LogLevel: logger.Silent})
	require.NoError(t, err)
	defer database.Close()

	require.NoError(t, database.Exec("CREATE TABLE sites (id integer primary key, name text)").Error)
	require.NoError(t, database.Exec("CREATE TABLE site_users (site_id integer, user_id integer)").Error)
	require.NoError(t, database.Exec("CREATE TABLE roles (id integer primary key, site_id integer)").Error)
	require.NoError(t, database.Exec("CREATE TABLE user_roles (user_id integer, role_id integer, site_id integer)").Error)
	require.NoError(t, database.Exec("INSERT INTO sites (id, name) VALUES (1, 'legacy')").Error)
	require.NoError(t, database.Exec("INSERT INTO site_users (site_id, user_id) VALUES (1, 7)").Error)
	require.NoError(t, database.Exec("INSERT INTO roles (id, site_id) VALUES (1, NULL), (2, 1)").Error)
	require.NoError(t, database.Exec("INSERT INTO user_roles (user_id, role_id, site_id) VALUES (7, 2, 1), (8, 1, NULL)").Error)

	status, err := InspectLegacyMultiSiteSchema(context.Background(), database)
	require.NoError(t, err)

	assert.Equal(t, LegacyTableStatus{Exists: true, Rows: 1}, status.Sites)
	assert.Equal(t, LegacyTableStatus{Exists: true, Rows: 1}, status.SiteUsers)
	assert.Equal(t, LegacyColumnStatus{Exists: true, NonNullRows: 1}, status.RoleSiteID)
	assert.Equal(t, LegacyColumnStatus{Exists: true, NonNullRows: 1}, status.UserRoleSiteID)
	assert.True(t, status.HasLegacyData)
	assert.Contains(t, status.Recommendation, "back up")

	assert.True(t, database.Migrator().HasTable("sites"))
	assert.True(t, database.Migrator().HasColumn("roles", "site_id"))
}

func TestInspectLegacyMultiSiteSchemaHandlesAbsentArtifacts(t *testing.T) {
	database, err := Init(InitOptions{DSN: ":memory:", LogLevel: logger.Silent})
	require.NoError(t, err)
	defer database.Close()

	status, err := InspectLegacyMultiSiteSchema(context.Background(), database)
	require.NoError(t, err)

	assert.False(t, status.Sites.Exists)
	assert.False(t, status.SiteUsers.Exists)
	assert.False(t, status.RoleSiteID.Exists)
	assert.False(t, status.UserRoleSiteID.Exists)
	assert.False(t, status.HasLegacyData)
}

func TestInitReadOnlyDoesNotCreateMissingSQLiteDatabaseOrDirectory(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "missing", "impress.db")

	database, err := InitReadOnly(InitOptions{DSN: databasePath, LogLevel: logger.Silent})
	assert.Nil(t, database)
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)

	_, statErr := os.Stat(databasePath)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(filepath.Dir(databasePath))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestInitReadOnlyForcesSQLiteModeRO(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "impress.db")
	writable, err := Init(InitOptions{DSN: databasePath, LogLevel: logger.Silent})
	require.NoError(t, err)
	require.NoError(t, writable.Exec("CREATE TABLE existing (id integer primary key)").Error)
	require.NoError(t, writable.Close())

	readOnly, err := InitReadOnly(InitOptions{
		DSN:      "file:" + databasePath + "?mode=rwc",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer readOnly.Close()

	err = readOnly.Exec("CREATE TABLE forbidden (id integer primary key)").Error
	require.Error(t, err)
	assert.False(t, readOnly.Migrator().HasTable("forbidden"))
	assert.True(t, readOnly.Migrator().HasTable("existing"))
}

func TestCurrentAutoMigratePreservesLegacySchemaAndSiteConfig(t *testing.T) {
	database, err := Init(InitOptions{DSN: ":memory:", LogLevel: logger.Silent})
	require.NoError(t, err)
	defer database.Close()

	require.NoError(t, database.AutoMigrate(
		&model.User{},
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
		&model.SiteConfig{},
	))
	require.NoError(t, database.Exec("CREATE TABLE sites (id integer primary key, name text)").Error)
	require.NoError(t, database.Exec("CREATE TABLE site_users (site_id integer, user_id integer)").Error)
	require.NoError(t, database.Exec("ALTER TABLE roles ADD COLUMN site_id integer").Error)
	require.NoError(t, database.Exec("ALTER TABLE user_roles ADD COLUMN site_id integer").Error)
	require.NoError(t, database.Exec("INSERT INTO sites (id, name) VALUES (1, 'legacy')").Error)

	require.NoError(t, NewMigrator(database).AutoMigrate(
		&model.User{},
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
		&model.SiteConfig{},
	))

	assert.True(t, database.Migrator().HasTable("sites"))
	assert.True(t, database.Migrator().HasTable("site_users"))
	assert.True(t, database.Migrator().HasColumn("roles", "site_id"))
	assert.True(t, database.Migrator().HasColumn("user_roles", "site_id"))

	var sites int64
	require.NoError(t, database.Table("sites").Count(&sites).Error)
	assert.EqualValues(t, 1, sites)

	config := &model.SiteConfig{Key: model.SiteConfigKeyFeatures}
	require.NoError(t, database.Create(config).Error)
	var reloaded model.SiteConfig
	require.NoError(t, database.Where("key = ?", model.SiteConfigKeyFeatures).First(&reloaded).Error)
	assert.Equal(t, model.SiteConfigKeyFeatures, reloaded.Key)
}
