package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"blotting-consultancy/internal/db"
)

func TestLegacySiteStatusJSONContract(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "legacy-status.db")
	database, err := openDatabase(dsn)
	require.NoError(t, err)
	require.NoError(t, database.Exec("CREATE TABLE sites (id integer primary key)").Error)
	require.NoError(t, database.Exec("CREATE TABLE site_users (site_id integer, user_id integer)").Error)
	require.NoError(t, database.Exec("CREATE TABLE roles (id integer primary key, site_id integer)").Error)
	require.NoError(t, database.Exec("CREATE TABLE user_roles (user_id integer, role_id integer, site_id integer)").Error)
	require.NoError(t, database.Exec("INSERT INTO sites (id) VALUES (1)").Error)
	require.NoError(t, database.Exec("INSERT INTO roles (id, site_id) VALUES (1, 9)").Error)
	require.NoError(t, database.Close())
	require.NoError(t, os.Chmod(dsn, 0o444))
	beforeBytes, err := os.ReadFile(dsn)
	require.NoError(t, err)
	beforeInfo, err := os.Stat(dsn)
	require.NoError(t, err)

	cmd := migrateCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--dsn", dsn, "legacy-site-status", "--json"})
	require.NoError(t, cmd.Execute())

	var status db.LegacyMultiSiteStatus
	require.NoError(t, json.Unmarshal(output.Bytes(), &status))
	assert.EqualValues(t, 1, status.Sites.Rows)
	assert.EqualValues(t, 1, status.RoleSiteID.NonNullRows)
	assert.True(t, status.HasLegacyData)
	assert.Contains(t, status.Recommendation, "back up")

	afterBytes, err := os.ReadFile(dsn)
	require.NoError(t, err)
	afterInfo, err := os.Stat(dsn)
	require.NoError(t, err)
	assert.Equal(t, beforeBytes, afterBytes, "read-only inspection must not change schema or data bytes")
	assert.Equal(t, beforeInfo.ModTime(), afterInfo.ModTime(), "read-only inspection must not change database mtime")
	assert.Equal(t, beforeInfo.Mode(), afterInfo.Mode(), "read-only inspection must not change database mode")
}

func TestLegacySiteStatusDoesNotCreateMissingSQLiteDatabase(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "missing", "impress.db")
	cmd := migrateCmd()
	cmd.SetArgs([]string{"--dsn", databasePath, "legacy-site-status"})

	require.Error(t, cmd.Execute())
	_, err := os.Stat(databasePath)
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Dir(databasePath))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestOpenDatabaseStillCreatesSQLiteDatabaseForMigrationCommands(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "created", "impress.db")
	database, err := openDatabase(databasePath)
	require.NoError(t, err)
	require.NoError(t, database.Close())

	info, err := os.Stat(databasePath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}
