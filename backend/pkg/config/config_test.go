package config

import (
	"os"
	"testing"
)

func TestLoad_WithAllRequiredVars(t *testing.T) {
	// Setup
	os.Setenv("PORT", "9000")
	os.Setenv("DB_DSN", "test.db")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	os.Setenv("ENV", "production")
	defer cleanupEnv()

	// Execute
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected Port=9000, got %d", cfg.Port)
	}
	if cfg.DBDSN != "test.db" {
		t.Errorf("expected DBDSN='test.db', got '%s'", cfg.DBDSN)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected JWTSecret='test-secret', got '%s'", cfg.JWTSecret)
	}
	if cfg.JWTRefreshSecret != "test-refresh-secret" {
		t.Errorf("expected JWTRefreshSecret='test-refresh-secret', got '%s'", cfg.JWTRefreshSecret)
	}
	if cfg.Env != "production" {
		t.Errorf("expected Env='production', got '%s'", cfg.Env)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	// Setup - only required vars
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	defer cleanupEnv()

	// Execute
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != 8088 {
		t.Errorf("expected default Port=8088, got %d", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected default Env='development', got '%s'", cfg.Env)
	}
	if cfg.DBDSN != defaultSQLiteDSN {
		t.Errorf("expected default DBDSN='%s', got '%s'", defaultSQLiteDSN, cfg.DBDSN)
	}
	if cfg.BackupDir != "./backups" {
		t.Errorf("expected default BackupDir='./backups', got '%s'", cfg.BackupDir)
	}
	if cfg.ThemeCatalogURL != "" {
		t.Errorf("expected empty ThemeCatalogURL by default, got %q", cfg.ThemeCatalogURL)
	}
	if len(cfg.ThemeUMDAllowHosts) == 0 {
		t.Fatal("expected default ThemeUMDAllowHosts to be populated")
	}
}

func TestLoad_ThemeCatalogEnv(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	os.Setenv("INKLESS_THEME_CATALOG_URL", "https://inkless.run/marketplace/v1/themes.json")
	os.Setenv("INKLESS_THEME_UMD_ALLOW_HOSTS", "cdn.example.com,github.com")
	defer cleanupEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.ThemeCatalogURL != "https://inkless.run/marketplace/v1/themes.json" {
		t.Fatalf("ThemeCatalogURL: got %q", cfg.ThemeCatalogURL)
	}
	if len(cfg.ThemeUMDAllowHosts) != 2 || cfg.ThemeUMDAllowHosts[0] != "cdn.example.com" {
		t.Fatalf("ThemeUMDAllowHosts: got %#v", cfg.ThemeUMDAllowHosts)
	}
}

func TestLoad_UsesDefaultSQLiteDSNWhenMissing(t *testing.T) {
	// Setup - DB_DSN intentionally missing
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	defer cleanupEnv()

	// Execute
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.DBDSN != defaultSQLiteDSN {
		t.Errorf("expected default DBDSN='%s', got '%s'", defaultSQLiteDSN, cfg.DBDSN)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	// Setup - missing JWT_SECRET
	os.Setenv("DB_DSN", "test.db")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	defer cleanupEnv()

	// Execute
	_, err := Load()

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	expectedMsg := "missing required environment variables: [JWT_SECRET]"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestLoad_MissingJWTRefreshSecret(t *testing.T) {
	// Setup - missing JWT_REFRESH_SECRET
	os.Setenv("DB_DSN", "test.db")
	os.Setenv("JWT_SECRET", "test-secret")
	defer cleanupEnv()

	// Execute
	_, err := Load()

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	expectedMsg := "missing required environment variables: [JWT_REFRESH_SECRET]"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestLoad_MissingMultipleVars(t *testing.T) {
	// Setup - missing all required vars
	defer cleanupEnv()

	// Execute
	_, err := Load()

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	// Should contain only JWT-related missing vars (DB_DSN has default)
	if err.Error() != "missing required environment variables: [JWT_SECRET JWT_REFRESH_SECRET]" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	// Setup - invalid PORT
	os.Setenv("PORT", "not-a-number")
	os.Setenv("DB_DSN", "test.db")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	defer cleanupEnv()

	// Execute
	_, err := Load()

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid PORT, got nil")
	}
}

// cleanupEnv clears all config-related environment variables
func cleanupEnv() {
	os.Unsetenv("PORT")
	os.Unsetenv("DB_DSN")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_REFRESH_SECRET")
	os.Unsetenv("ENV")
	os.Unsetenv("BACKUP_DIR")
	os.Unsetenv("INKLESS_THEME_CATALOG_URL")
	os.Unsetenv("INKLESS_THEME_UMD_ALLOW_HOSTS")
}
