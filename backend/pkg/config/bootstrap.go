package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
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

	return cfg, nil
}

func ephemeralSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "bootstrap-" + hex.EncodeToString(buf), nil
}
