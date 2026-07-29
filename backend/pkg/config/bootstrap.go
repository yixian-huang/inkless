package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
)

// LoadResult contains configuration plus bootstrap metadata.
type LoadResult struct {
	Config           *Config
	BootstrapMode    bool
	EnvSecretsLoaded bool
}

// LoadWithBootstrap loads configuration, allowing ephemeral JWT secrets when
// SETUP_BOOTSTRAP=true or JWT env vars are missing.
func LoadWithBootstrap() (*LoadResult, error) {
	forceBootstrap := strings.EqualFold(strings.TrimSpace(os.Getenv("SETUP_BOOTSTRAP")), "true")
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	jwtRefresh := strings.TrimSpace(os.Getenv("JWT_REFRESH_SECRET"))
	secretsFromEnv := jwtSecret != "" && jwtRefresh != ""

	if secretsFromEnv && !forceBootstrap {
		cfg, err := Load()
		if err != nil {
			return nil, err
		}
		return &LoadResult{
			Config:           cfg,
			BootstrapMode:    false,
			EnvSecretsLoaded: true,
		}, nil
	}

	cfg, err := loadBase()
	if err != nil {
		return nil, err
	}

	if secretsFromEnv {
		cfg.JWTSecret = jwtSecret
		cfg.JWTRefreshSecret = jwtRefresh
		return &LoadResult{
			Config:           cfg,
			BootstrapMode:    forceBootstrap,
			EnvSecretsLoaded: true,
		}, nil
	}

	secret, err := ephemeralSecret()
	if err != nil {
		return nil, fmt.Errorf("generate jwt secret: %w", err)
	}
	refresh, err := ephemeralSecret()
	if err != nil {
		return nil, fmt.Errorf("generate jwt refresh secret: %w", err)
	}
	cfg.JWTSecret = secret
	cfg.JWTRefreshSecret = refresh

	return &LoadResult{
		Config:           cfg,
		BootstrapMode:    true,
		EnvSecretsLoaded: false,
	}, nil
}

// loadBase reads optional env vars without requiring JWT secrets.
func loadBase() (*Config, error) {
	cfg := &Config{}

	port, err := ParsePortString(os.Getenv("PORT"))
	if err != nil {
		return nil, err
	}
	cfg.Port = port

	cfg.DBDSN = strings.TrimSpace(os.Getenv("DB_DSN"))
	if cfg.DBDSN == "" {
		cfg.DBDSN = defaultSQLiteDSN
	}

	cfg.Env = strings.TrimSpace(os.Getenv("ENV"))
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	corsAllowedOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if corsAllowedOrigins != "" {
		cfg.CORSAllowedOrigins = splitAndTrim(corsAllowedOrigins)
	} else if cfg.Env == "development" {
		cfg.CORSAllowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
		}
	}

	cfg.UploadDir = strings.TrimSpace(os.Getenv("UPLOAD_DIR"))
	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads"
	}
	cfg.BackupDir = strings.TrimSpace(os.Getenv("BACKUP_DIR"))
	if cfg.BackupDir == "" {
		cfg.BackupDir = "./backups"
	}

	cfg.BaseURL = strings.TrimSpace(os.Getenv("BASE_URL"))
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://www.example.com"
	}

	cfg.FrontendDir = strings.TrimSpace(os.Getenv("FRONTEND_DIR"))

	cfg.PluginDir = strings.TrimSpace(os.Getenv("PLUGIN_DIR"))
	if cfg.PluginDir == "" {
		cfg.PluginDir = "./plugins"
	}
	cfg.PluginDataDir = strings.TrimSpace(os.Getenv("PLUGIN_DATA_DIR"))
	if cfg.PluginDataDir == "" {
		cfg.PluginDataDir = "./data/plugins"
	}
	cfg.ExternalPlugins = strings.EqualFold(
		strings.TrimSpace(os.Getenv("ENABLE_EXTERNAL_PLUGINS")),
		"true",
	)

	// Default true for backward-compatible dual-track reads. Disable after
	// content_documents → unified_pages migration is complete:
	// LEGACY_CONTENT_DOC_FALLBACK=0|false|off
	cfg.LegacyContentDocFallback = parseBoolDefaultTrue(os.Getenv("LEGACY_CONTENT_DOC_FALLBACK"))

	// Official theme catalog (Phase A). See docs/design-official-extension-store-phase-a.md
	cfg.ThemeCatalogURL = strings.TrimSpace(os.Getenv("INKLESS_THEME_CATALOG_URL"))
	cfg.ThemeUMDAllowHosts = themecatalog.ParseAllowHosts(os.Getenv("INKLESS_THEME_UMD_ALLOW_HOSTS"))

	// Host self-update (H0/H1). See docs/design-host-self-update-mvp.md
	cfg.SelfUpdateEnabled = parseBoolDefaultFalse(os.Getenv("INKLESS_SELF_UPDATE_ENABLED"))
	cfg.SelfUpdateReleaseRoot = strings.TrimSpace(os.Getenv("INKLESS_RELEASE_ROOT"))
	cfg.SelfUpdateSystemdUnit = strings.TrimSpace(os.Getenv("INKLESS_SYSTEMD_UNIT"))
	cfg.SelfUpdateRepo = strings.TrimSpace(os.Getenv("INKLESS_UPDATE_REPO"))
	if cfg.SelfUpdateRepo == "" {
		cfg.SelfUpdateRepo = "yixian-huang/inkless"
	}
	cfg.SelfUpdateChannel = strings.ToLower(strings.TrimSpace(os.Getenv("INKLESS_UPDATE_CHANNEL")))
	if cfg.SelfUpdateChannel == "" {
		cfg.SelfUpdateChannel = "stable"
	}
	cfg.SelfUpdateManifestURL = strings.TrimSpace(os.Getenv("INKLESS_UPDATE_MANIFEST_URL"))
	cfg.SelfUpdateAPIBase = strings.TrimSpace(os.Getenv("INKLESS_UPDATE_API_BASE"))
	if cfg.SelfUpdateAPIBase == "" {
		cfg.SelfUpdateAPIBase = "https://api.github.com"
	}
	cfg.SelfUpdateCheckTTLSec = parsePositiveInt(os.Getenv("INKLESS_UPDATE_CHECK_TTL_SEC"), 900)
	if hosts := strings.TrimSpace(os.Getenv("INKLESS_UPDATE_ALLOW_HOSTS")); hosts != "" {
		cfg.SelfUpdateAllowHosts = splitAndTrim(hosts)
	}
	cfg.SelfUpdateGitHubToken = strings.TrimSpace(os.Getenv("INKLESS_GITHUB_TOKEN"))
	cfg.SelfUpdateHealthURL = strings.TrimSpace(os.Getenv("INKLESS_UPDATE_HEALTH_URL"))
	cfg.SelfUpdateHealthTimeout = parsePositiveInt(os.Getenv("INKLESS_UPDATE_HEALTH_TIMEOUT_SEC"), 60)

	return cfg, nil
}

func parseBoolDefaultFalse(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parsePositiveInt(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	var n int
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}

// parseBoolDefaultTrue treats empty as true; 0/false/off/no as false.
func parseBoolDefaultTrue(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func ephemeralSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "bootstrap-" + hex.EncodeToString(buf), nil
}
