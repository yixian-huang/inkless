package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithBootstrap_GeneratesEphemeralJWT(t *testing.T) {
	cleanupBootstrapEnv()
	os.Setenv("SETUP_BOOTSTRAP", "true")
	defer cleanupBootstrapEnv()

	result, err := LoadWithBootstrap()
	require.NoError(t, err)
	assert.True(t, result.BootstrapMode)
	assert.False(t, result.EnvSecretsLoaded)
	assert.NotEmpty(t, result.Config.JWTSecret)
	assert.NotEmpty(t, result.Config.JWTRefreshSecret)
}

func TestLoadWithBootstrap_UsesEnvSecretsWhenPresent(t *testing.T) {
	cleanupBootstrapEnv()
	os.Setenv("JWT_SECRET", "from-env-secret")
	os.Setenv("JWT_REFRESH_SECRET", "from-env-refresh-secret")
	defer cleanupBootstrapEnv()

	result, err := LoadWithBootstrap()
	require.NoError(t, err)
	assert.False(t, result.BootstrapMode)
	assert.True(t, result.EnvSecretsLoaded)
	assert.Equal(t, "from-env-secret", result.Config.JWTSecret)
}

func TestExternalPluginsAreDisabledByDefault(t *testing.T) {
	cleanupBootstrapEnv()
	defer cleanupBootstrapEnv()

	cfg, err := loadBase()
	require.NoError(t, err)
	assert.False(t, cfg.ExternalPlugins)
}

func TestExternalPluginsRequireExplicitEnable(t *testing.T) {
	cleanupBootstrapEnv()
	os.Setenv("ENABLE_EXTERNAL_PLUGINS", "true")
	defer cleanupBootstrapEnv()

	cfg, err := loadBase()
	require.NoError(t, err)
	assert.True(t, cfg.ExternalPlugins)
}

func TestBackupDirCanBeConfiguredPerInstance(t *testing.T) {
	cleanupBootstrapEnv()
	os.Setenv("BACKUP_DIR", "/srv/impress-a/backups")
	defer cleanupBootstrapEnv()

	cfg, err := loadBase()
	require.NoError(t, err)
	assert.Equal(t, "/srv/impress-a/backups", cfg.BackupDir)
}

func TestWriteEnvFile_CreatesEnvAndDirs(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteEnvFile(dir, EnvFileParams{
		Port:             9090,
		DBDSN:            "file:" + filepath.Join(dir, "data", "test.db") + "?cache=shared&mode=rwc",
		JWTSecret:        "secret",
		JWTRefreshSecret: "refresh",
	})
	require.NoError(t, err)
	assert.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "PORT=9090")
	assert.Contains(t, content, "JWT_SECRET=secret")
	assert.Contains(t, content, "BACKUP_DIR=./backups")
	assert.DirExists(t, filepath.Join(dir, "backups"))
}

func cleanupBootstrapEnv() {
	os.Unsetenv("SETUP_BOOTSTRAP")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_REFRESH_SECRET")
	os.Unsetenv("PORT")
	os.Unsetenv("DB_DSN")
	os.Unsetenv("ENV")
	os.Unsetenv("ENABLE_EXTERNAL_PLUGINS")
	os.Unsetenv("BACKUP_DIR")
}
