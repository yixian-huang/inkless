package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration values for the application
type Config struct {
	Port               int
	DBDSN              string
	JWTSecret          string
	JWTRefreshSecret   string
	Env                string
	CORSAllowedOrigins []string
	UploadDir          string
	BackupDir          string
	BaseURL            string
	FrontendDir        string
	PluginDir          string
	PluginDataDir      string
	ExternalPlugins    bool
	// LegacyContentDocFallback merges content_documents into unified page
	// public reads when true (default). Set LEGACY_CONTENT_DOC_FALLBACK=0
	// after migration to skip the dual-track merge when unified has content.
	LegacyContentDocFallback bool

	// ThemeCatalogURL is the optional remote official themes index
	// (INKLESS_THEME_CATALOG_URL). Empty → use embedded fallback only until
	// remote fetch is wired (Phase A catalog service).
	ThemeCatalogURL string
	// ThemeUMDAllowHosts is the HTTPS host allowlist for marketplace/official
	// theme UMD URLs (INKLESS_THEME_UMD_ALLOW_HOSTS, comma-separated).
	// Empty env → themecatalog.DefaultUMDAllowHosts.
	ThemeUMDAllowHosts []string
}

const defaultSQLiteDSN = "file:./data/inkless.db?cache=shared&mode=rwc"

// Load reads configuration from environment variables with validation and defaults
func Load() (*Config, error) {
	cfg, err := loadBase()
	if err != nil {
		return nil, err
	}

	var missingVars []string
	cfg.JWTSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if cfg.JWTSecret == "" {
		missingVars = append(missingVars, "JWT_SECRET")
	}
	cfg.JWTRefreshSecret = strings.TrimSpace(os.Getenv("JWT_REFRESH_SECRET"))
	if cfg.JWTRefreshSecret == "" {
		missingVars = append(missingVars, "JWT_REFRESH_SECRET")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return cfg, nil
}

func splitAndTrim(csv string) []string {
	items := strings.Split(csv, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
