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
}

const defaultSQLiteDSN = "file:./data/impress.db?cache=shared&mode=rwc"

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
