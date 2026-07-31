package setup

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/seed"
	"github.com/yixian-huang/inkless/backend/pkg/auth"
	"github.com/yixian-huang/inkless/backend/pkg/config"
	"gorm.io/gorm"
)

var (
	ErrAlreadyCompleted = errors.New("setup already completed")
	ErrInvalidInput     = errors.New("invalid setup input")
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// AdminInput is the administrator account for first-time setup.
type AdminInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SiteInput is basic site metadata collected during setup.
type SiteInput struct {
	Name          map[string]string `json:"name"`
	DefaultLocale string            `json:"defaultLocale"`
}

// CompleteInput is the payload for POST /setup/complete.
type CompleteInput struct {
	Admin    AdminInput `json:"admin"`
	Site     SiteInput  `json:"site"`
	SeedMode string     `json:"seedMode"`
}

// Status describes whether the instance has completed web setup.
type Status struct {
	Installed        bool   `json:"installed"`
	DatabaseType     string `json:"databaseType"`
	BootstrapMode    bool   `json:"bootstrapMode"`
	NeedsEnvConfig   bool   `json:"needsEnvConfig"`
	EnvSecretsLoaded bool   `json:"envSecretsLoaded"`
	EnvFilePath      string `json:"envFilePath"`
	ServerPort       int    `json:"serverPort"`
}

// ServiceOptions configures bootstrap-aware setup behavior.
type ServiceOptions struct {
	BootstrapMode    bool
	EnvSecretsLoaded bool
	DatabaseType     string
	EnvFilePath      string
	ServerPort       int
	WorkingDir       string
}

// SeederFactory builds a seeder bound to the given database handle.
type SeederFactory func(db *gorm.DB) *seed.Seeder

// Service handles first-run installation via the web wizard.
type Service struct {
	db            *gorm.DB
	userRepo      repository.UserRepository
	siteCfgRepo   repository.SiteConfigRepository
	seederFactory SeederFactory
	seedRBAC      func(ctx context.Context, roleRepo repository.RoleRepository) error
	opts          ServiceOptions
}

// NewService creates a setup service.
func NewService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	siteCfgRepo repository.SiteConfigRepository,
	seederFactory SeederFactory,
	seedRBAC func(ctx context.Context, roleRepo repository.RoleRepository) error,
	opts ServiceOptions,
) *Service {
	if opts.EnvFilePath == "" {
		opts.EnvFilePath = config.DefaultEnvFilePath()
	}
	if opts.WorkingDir == "" {
		opts.WorkingDir = WorkingDirectory()
	}
	if opts.ServerPort == 0 {
		opts.ServerPort = 8088
	}
	return &Service{
		db:            db,
		userRepo:      userRepo,
		siteCfgRepo:   siteCfgRepo,
		seederFactory: seederFactory,
		seedRBAC:      seedRBAC,
		opts:          opts,
	}
}

// IsInstalled reports whether setup has already been completed.
func (s *Service) IsInstalled(ctx context.Context) (bool, error) {
	if installed, err := s.hasSystemInstallRecord(ctx); err != nil {
		return false, err
	} else if installed {
		return true, nil
	}

	// Legacy: deployments seeded before the web wizard only have a super admin row.
	count, err := s.userRepo.CountSuperAdmins(ctx)
	if err != nil {
		return false, err
	}
	if count > 0 {
		log.Println("setup: legacy super-admin detected without system install record")
		return true, nil
	}
	return false, nil
}

// GetStatus returns install state plus database type hint for the wizard UI.
func (s *Service) GetStatus(ctx context.Context) (*Status, error) {
	installed, err := s.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	return &Status{
		Installed:        installed,
		DatabaseType:     s.opts.DatabaseType,
		BootstrapMode:    s.opts.BootstrapMode,
		NeedsEnvConfig:   s.NeedsEnvConfig(),
		EnvSecretsLoaded: s.opts.EnvSecretsLoaded,
		EnvFilePath:      s.opts.EnvFilePath,
		ServerPort:       s.opts.ServerPort,
	}, nil
}

// NeedsEnvConfig reports whether persisted env secrets are required before install can finish.
func (s *Service) NeedsEnvConfig() bool {
	return !s.opts.EnvSecretsLoaded
}

// BootstrapMode reports whether the server started without persisted JWT secrets.
func (s *Service) BootstrapMode() bool {
	return s.opts.BootstrapMode
}

// ServerPort returns the API port exposed to the setup wizard.
func (s *Service) ServerPort() int {
	return s.opts.ServerPort
}

// AllowsEnvConfiguration reports whether bootstrap env endpoints should be accepted.
func (s *Service) AllowsEnvConfiguration(ctx context.Context) (bool, error) {
	installed, err := s.IsInstalled(ctx)
	if err != nil {
		return false, err
	}
	return !installed && s.NeedsEnvConfig(), nil
}

// Complete runs the one-shot installation flow.
func (s *Service) Complete(ctx context.Context, in CompleteInput) error {
	installed, err := s.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		return ErrAlreadyCompleted
	}
	if s.NeedsEnvConfig() {
		return fmt.Errorf("%w: load persisted environment secrets and restart the server first", ErrInvalidInput)
	}
	if err := validateInput(in); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	username := strings.TrimSpace(in.Admin.Username)
	existing, _ := s.userRepo.FindByUsername(ctx, username)
	if existing != nil {
		return fmt.Errorf("%w: username already exists", ErrInvalidInput)
	}

	hashedPassword, err := auth.HashPassword(in.Admin.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	seedMode := strings.ToLower(strings.TrimSpace(in.SeedMode))
	if seedMode != "demo" {
		seedMode = "blank"
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		seeder := s.seederFactory(tx)
		siteCfgRepo := repository.NewGormSiteConfigRepository(tx)
		userRepo := repository.NewGormUserRepository(tx)

		switch seedMode {
		case "demo":
			if err := seeder.DemoSiteSeedContent(ctx); err != nil {
				return fmt.Errorf("demo seed: %w", err)
			}
		default:
			if err := seeder.BlankSiteSeedContent(ctx); err != nil {
				return fmt.Errorf("blank seed: %w", err)
			}
		}

		if err := applyGlobalSiteName(ctx, siteCfgRepo, in.Site); err != nil {
			return fmt.Errorf("apply site name: %w", err)
		}

		user := &model.User{
			Username:     username,
			PasswordHash: hashedPassword,
			Role:         model.RoleAdmin,
			IsSuperAdmin: true,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("create admin: %w", err)
		}

		installCfg := model.JSONMap{
			"installed":        true,
			"installedAt":      time.Now().UTC().Format(time.RFC3339),
			"seedMode":         seedMode,
			"installerVersion": "1",
		}
		row := &model.SiteConfig{
			Key:              model.SiteConfigKeySystem,
			DraftConfig:      installCfg,
			DraftVersion:     1,
			PublishedConfig:  installCfg,
			PublishedVersion: 1,
		}
		if err := siteCfgRepo.Upsert(ctx, row); err != nil {
			return fmt.Errorf("write install record: %w", err)
		}

		if s.seedRBAC != nil {
			roleRepo := repository.NewGormRoleRepository(tx)
			if err := s.seedRBAC(ctx, roleRepo); err != nil {
				return fmt.Errorf("seed rbac: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) hasSystemInstallRecord(ctx context.Context) (bool, error) {
	sc, err := s.siteCfgRepo.FindByKey(ctx, model.SiteConfigKeySystem)
	if err != nil {
		if repository.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if sc == nil || sc.ID == 0 {
		return false, nil
	}
	installed, _ := sc.PublishedConfig["installed"].(bool)
	return installed, nil
}

func validateInput(in CompleteInput) error {
	username := strings.TrimSpace(in.Admin.Username)
	if !usernamePattern.MatchString(username) {
		return errors.New("username must be 3-32 characters (letters, digits, underscore)")
	}
	if err := validatePassword(in.Admin.Password); err != nil {
		return err
	}

	nameZH := strings.TrimSpace(in.Site.Name["zh"])
	nameEN := strings.TrimSpace(in.Site.Name["en"])
	if nameZH == "" && nameEN == "" {
		return errors.New("site name is required")
	}

	locale := strings.TrimSpace(in.Site.DefaultLocale)
	if locale != "" && locale != "zh" && locale != "en" {
		return errors.New("defaultLocale must be zh or en")
	}

	mode := strings.ToLower(strings.TrimSpace(in.SeedMode))
	if mode != "" && mode != "blank" && mode != "demo" {
		return errors.New("seedMode must be blank or demo")
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must contain letters and digits")
	}
	return nil
}

func applyGlobalSiteName(ctx context.Context, siteCfgRepo repository.SiteConfigRepository, site SiteInput) error {
	if siteCfgRepo == nil {
		return nil
	}
	sc, err := siteCfgRepo.FindByKey(ctx, model.SiteConfigKeyGlobal)
	if err != nil && !repository.IsNotFound(err) && (sc == nil || sc.ID == 0) {
		// treat missing row as create-below
		sc = nil
	}
	if sc == nil || sc.ID == 0 {
		sc = &model.SiteConfig{
			Key:              model.SiteConfigKeyGlobal,
			DraftConfig:      model.JSONMap{},
			DraftVersion:     1,
			PublishedConfig:  model.JSONMap{},
			PublishedVersion: 1,
		}
	}

	cfg := sc.PublishedConfig
	if cfg == nil {
		cfg = model.JSONMap{}
	}

	identity, _ := cfg["identity"].(model.JSONMap)
	if identity == nil {
		identity = model.JSONMap{}
	}

	nameMap := model.JSONMap{}
	if v := strings.TrimSpace(site.Name["zh"]); v != "" {
		nameMap["zh"] = v
	}
	if v := strings.TrimSpace(site.Name["en"]); v != "" {
		nameMap["en"] = v
	}
	if len(nameMap) > 0 {
		identity["name"] = nameMap
	}

	locale := strings.TrimSpace(site.DefaultLocale)
	if locale == "" {
		locale = "zh"
	}
	identity["defaultLocale"] = locale
	switch locale {
	case "en":
		identity["localeMode"] = "mono-en"
	case "zh":
		identity["localeMode"] = "mono-zh"
	default:
		identity["localeMode"] = "bilingual"
	}

	cfg["identity"] = identity
	sc.DraftConfig = cfg
	sc.PublishedConfig = cfg
	if sc.DraftVersion < 1 {
		sc.DraftVersion = 1
	}
	if sc.PublishedVersion < 1 {
		sc.PublishedVersion = 1
	}
	sc.Key = model.SiteConfigKeyGlobal
	return siteCfgRepo.Upsert(ctx, sc)
}
