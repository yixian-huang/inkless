package seed

import (
	"context"
	"log"
	"strings"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/auth"
	"github.com/yixian-huang/inkless/backend/pkg/brand"
)

// Seeder handles idempotent seeding of initial data
type Seeder struct {
	userRepo           repository.UserRepository
	contentRepo        repository.ContentDocumentRepository
	installedThemeRepo repository.InstalledThemeRepository
	themePageService   *service.ThemePageService
	unifiedPageRepo    repository.UnifiedPageRepository
	templateRepo       repository.PageTemplateRepository
	siteCfgRepo        repository.SiteConfigRepository
}

// NewSeeder creates a new seeder instance
func NewSeeder(
	userRepo repository.UserRepository,
	contentRepo repository.ContentDocumentRepository,
	installedThemeRepo repository.InstalledThemeRepository,
	themePageService *service.ThemePageService,
	unifiedPageRepo repository.UnifiedPageRepository,
	templateRepo repository.PageTemplateRepository,
	siteCfgRepo repository.SiteConfigRepository,
) *Seeder {
	return &Seeder{
		userRepo:           userRepo,
		contentRepo:        contentRepo,
		installedThemeRepo: installedThemeRepo,
		themePageService:   themePageService,
		unifiedPageRepo:    unifiedPageRepo,
		templateRepo:       templateRepo,
		siteCfgRepo:        siteCfgRepo,
	}
}

// SeedAll seeds all required initial data idempotently
func (s *Seeder) SeedAll(ctx context.Context) error {
	log.Println("Starting seed process...")

	if err := s.SeedUsers(ctx); err != nil {
		return err
	}

	if err := s.SeedContentDocuments(ctx); err != nil {
		return err
	}

	if err := s.SeedUnifiedPages(ctx); err != nil {
		return err
	}

	if err := s.SeedInstalledThemes(ctx); err != nil {
		return err
	}

	if err := s.SeedThemePages(ctx); err != nil {
		return err
	}

	log.Println("Seed process completed successfully")
	return nil
}

// SeedUsers creates default admin and editor users if they don't exist
func (s *Seeder) SeedUsers(ctx context.Context) error {
	defaultUsers := []struct {
		Username     string
		Password     string
		Role         model.Role
		IsSuperAdmin bool
	}{
		{Username: "admin", Password: "admin123", Role: model.RoleAdmin, IsSuperAdmin: true},
		{Username: "editor", Password: "editor123", Role: model.RoleEditor},
	}

	for _, userData := range defaultUsers {
		// Check if user already exists
		existingUser, err := s.userRepo.FindByUsername(ctx, userData.Username)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}

		if existingUser != nil {
			log.Printf("User %s already exists, skipping", userData.Username)
			continue
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(userData.Password)
		if err != nil {
			return err
		}

		// Create new user
		user := &model.User{
			Username:     userData.Username,
			PasswordHash: hashedPassword,
			Role:         userData.Role,
			IsSuperAdmin: userData.IsSuperAdmin,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}

		log.Printf("Created user: %s with role: %s", userData.Username, userData.Role)
	}

	return nil
}

// SeedContentDocuments creates initial content documents for all page keys if they don't exist
func (s *Seeder) SeedContentDocuments(ctx context.Context) error {
	for _, pageKey := range model.ValidPageKeys {
		// Check if content document already exists
		existingDoc, err := s.contentRepo.FindByPageKey(ctx, pageKey)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}

		if existingDoc != nil {
			log.Printf("Content document for %s already exists, skipping", pageKey)
			continue
		}

		// Create initial empty content document
		doc := &model.ContentDocument{
			PageKey:          pageKey,
			DraftConfig:      getInitialConfig(pageKey),
			DraftVersion:     1,
			PublishedConfig:  getInitialConfig(pageKey),
			PublishedVersion: 1,
		}

		if err := s.contentRepo.Create(ctx, doc); err != nil {
			return err
		}

		log.Printf("Created content document for page: %s", pageKey)
	}

	return nil
}

// builtinPageDefs defines the 7 builtin pages with deterministic IDs.
var builtinPageDefs = []struct {
	ID     uint
	Slug   string
	ZhName string
	EnName string
}{
	{1, "home", "首页", "Home"},
	{2, "about", "关于我们", "About Us"},
	{3, "advantages", "我们的优势", "Our Advantages"},
	{4, "core-services", "核心服务", "Core Services"},
	{5, "cases", "案例展示", "Case Studies"},
	{6, "experts", "专家团队", "Our Experts"},
	{7, "contact", "联系我们", "Contact Us"},
}

// SeedUnifiedPages creates unified pages and page templates for the 7 builtin pages.
// Converts the same seed data from content_documents into sections format.
func (s *Seeder) SeedUnifiedPages(ctx context.Context) error {
	if s.unifiedPageRepo == nil {
		return nil
	}

	for _, def := range builtinPageDefs {
		// Check if unified page already exists
		existing, err := s.unifiedPageRepo.FindBySlug(ctx, def.Slug)
		if err == nil && existing != nil {
			log.Printf("Unified page %s already exists, skipping", def.Slug)
			continue
		}

		// Convert seed config to sections format
		pageKey := model.PageKey(def.Slug)
		flatConfig := getInitialConfig(pageKey)
		sectionsConfig := service.ConvertContentDocToSections(def.Slug, flatConfig)

		// Create page template
		if s.templateRepo != nil {
			templateKey := "builtin-" + def.Slug
			existingTpl, tplErr := s.templateRepo.FindByKey(ctx, templateKey)
			if tplErr != nil || existingTpl == nil {
				tpl := &model.PageTemplate{
					ID:       def.ID,
					Key:      templateKey,
					NameZh:   def.ZhName,
					NameEn:   def.EnName,
					Category: "builtin",
					Config:   sectionsConfig,
				}
				if err := s.templateRepo.Create(ctx, tpl); err != nil {
					log.Printf("Warning: failed to create page template %s: %v", templateKey, err)
				}
			}
		}

		// Create unified page
		templateID := def.ID
		pubConfig := model.NullableJSONMap(sectionsConfig)
		page := &model.UnifiedPage{
			ID:               def.ID,
			Slug:             def.Slug,
			ZhTitle:          def.ZhName,
			EnTitle:          def.EnName,
			Mode:             model.PageModeTemplate,
			TemplateID:       &templateID,
			DraftConfig:      sectionsConfig,
			DraftVersion:     1,
			PublishedConfig:  pubConfig,
			PublishedVersion: 1,
			Status:           "published",
			SortOrder:        int(def.ID),
			ShowInNav:        true,
		}

		if err := s.unifiedPageRepo.Create(ctx, page); err != nil {
			return err
		}

		log.Printf("Created unified page: %s (ID=%d)", def.Slug, def.ID)
	}

	return nil
}

// SeedInstalledThemes creates built-in themes if they don't exist
func (s *Seeder) SeedInstalledThemes(ctx context.Context) error {
	if err := s.ensureInstalledTheme(ctx, &model.InstalledTheme{
		ThemeID:     builtinthemes.CorporateClassic,
		Name:        "Corporate Classic",
		NameZh:      "企业经典",
		Description: "专业企业官网，含首页、关于、优势、服务、案例、专家、联系",
		Author:      brand.ProductName,
		Version:     "1.0.0",
		Source:      "built-in",
		IsActive:    true,
		Preview:     "linear-gradient(135deg, #1a5f8f 0%, #8bc34a 100%)",
	}); err != nil {
		return err
	}

	if err := s.ensureInstalledTheme(ctx, &model.InstalledTheme{
		ThemeID:     builtinthemes.BlogFirst,
		Name:        "Blog First",
		NameZh:      "博客优先",
		Description: "极简个人博客，首页展示作者介绍与最近文章",
		Author:      brand.ProductName,
		Version:     "1.0.0",
		Source:      "built-in",
		IsActive:    false,
		Preview:     "linear-gradient(135deg, #1e40af 0%, #64748b 100%)",
	}); err != nil {
		return err
	}

	if err := s.ensureInstalledTheme(ctx, &model.InstalledTheme{
		ThemeID:     builtinthemes.ProductFirst,
		Name:        "Product First",
		NameZh:      "产品优先",
		Description: "软件产品介绍站：主视觉、能力、安装引导、可选更新日志",
		Author:      brand.ProductName,
		Version:     "0.1.0",
		Source:      "built-in",
		IsActive:    false,
		Preview:     "linear-gradient(135deg, #111827 0%, #14b8a6 100%)",
	}); err != nil {
		return err
	}

	if err := s.ensureInstalledTheme(ctx, &model.InstalledTheme{
		ThemeID:     builtinthemes.EditorialFirm,
		Name:        "Editorial Firm",
		NameZh:      "编辑机构",
		Description: "杂志气质机构官网：首页、关于、服务、联系",
		Author:      brand.ProductName,
		Version:     "0.1.0",
		Source:      "built-in",
		IsActive:    false,
		Preview:     "linear-gradient(135deg, #111111 0%, #C45C26 100%)",
	}); err != nil {
		return err
	}

	return s.seedMinimalStarterTheme(ctx)
}

func (s *Seeder) seedMinimalStarterTheme(ctx context.Context) error {
	return s.ensureInstalledTheme(ctx, &model.InstalledTheme{
		ThemeID:     builtinthemes.MinimalStarter,
		Name:        "Minimal Starter",
		NameZh:      "极简起步",
		Description: "最简内置主题，演示第三方主题扩展路径",
		Author:      brand.ProductName,
		Version:     "1.0.0",
		Source:      "built-in",
		IsActive:    false,
		Preview:     "linear-gradient(135deg, #374151 0%, #9ca3af 100%)",
	})
}

func (s *Seeder) ensureInstalledTheme(ctx context.Context, theme *model.InstalledTheme) error {
	existing, err := s.installedThemeRepo.FindByThemeID(ctx, theme.ThemeID)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}

	if existing != nil {
		log.Printf("InstalledTheme %s already exists, skipping", theme.ThemeID)
		return nil
	}

	if err := s.installedThemeRepo.Create(ctx, theme); err != nil {
		return err
	}

	log.Printf("Created InstalledTheme: %s (active=%v)", theme.ThemeID, theme.IsActive)
	return nil
}

// SeedThemePages seeds pages for the active theme
func (s *Seeder) SeedThemePages(ctx context.Context) error {
	// Find the active theme
	activeTheme, err := s.installedThemeRepo.FindActive(ctx)
	if err != nil {
		log.Println("No active theme found, skipping theme page seeding")
		return nil
	}

	log.Printf("Seeding pages for active theme: %s", activeTheme.ThemeID)
	return s.themePageService.SeedThemePages(ctx, activeTheme.ThemeID)
}

// DemoSiteSeed runs the full consultancy demo data seed.
// Equivalent to the legacy SeedAll().
func (s *Seeder) DemoSiteSeed(ctx context.Context) error {
	return s.SeedAll(ctx)
}

// DemoSiteSeedContent seeds demo content without creating default users.
func (s *Seeder) DemoSiteSeedContent(ctx context.Context) error {
	log.Println("Starting demo-site content seed (no users)...")
	if err := s.SeedContentDocuments(ctx); err != nil {
		return err
	}
	if err := s.SeedUnifiedPages(ctx); err != nil {
		return err
	}
	if err := s.SeedInstalledThemes(ctx); err != nil {
		return err
	}
	if err := s.SeedThemePages(ctx); err != nil {
		return err
	}
	log.Println("Demo-site content seed completed")
	return nil
}

// BlankSiteSeed inserts the minimum required for a fresh personal-blog site:
// one admin user, the global content document with default SiteConfigGlobal
// payload, and (if siteCfgRepo provided) the features site_config with
// personal-blog defaults. No articles, no media, no demo data.
func (s *Seeder) BlankSiteSeed(ctx context.Context) error {
	log.Println("Starting blank-site seed...")
	if err := s.SeedUsers(ctx); err != nil {
		return err
	}
	return s.BlankSiteSeedContent(ctx)
}

// BlankSiteSeedContent seeds a blank site without creating default users.
func (s *Seeder) BlankSiteSeedContent(ctx context.Context) error {
	log.Println("Starting blank-site content seed (no users)...")

	// Ensure a "global" content_document exists with personal-blog defaults.
	existing, err := s.contentRepo.FindByPageKey(ctx, model.PageKeyGlobal)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if existing == nil {
		doc := &model.ContentDocument{
			PageKey:          model.PageKeyGlobal,
			DraftConfig:      blankGlobalConfig(),
			DraftVersion:     1,
			PublishedConfig:  blankGlobalConfig(),
			PublishedVersion: 1,
		}
		if err := s.contentRepo.Create(ctx, doc); err != nil {
			return err
		}
		log.Println("Created blank global content document")
	}

	// Seed site_configs.features with personal-blog defaults so the gates
	// in PR-4 default to "consultancy pages off". Without an explicit record,
	// the frontend treats missing features as "all on" (old-deploy compat).
	if s.siteCfgRepo != nil {
		// GormSiteConfigRepository.FindByKey returns (&sc, err) with a zero-value
		// struct (ID == 0) when the row is missing — the pointer is NEVER nil.
		// Check `existing.ID == 0` (or err) instead of `existing == nil`.
		featuresExisting, ferr := s.siteCfgRepo.FindByKey(ctx, model.SiteConfigKeyFeatures)
		if ferr != nil && !strings.Contains(ferr.Error(), "not found") && !strings.Contains(ferr.Error(), "record not found") {
			return ferr
		}
		if featuresExisting == nil || featuresExisting.ID == 0 {
			cfg := blankFeaturesConfig()
			row := &model.SiteConfig{
				Key:              model.SiteConfigKeyFeatures,
				DraftConfig:      cfg,
				DraftVersion:     1,
				PublishedConfig:  cfg,
				PublishedVersion: 1,
			}
			if err := s.siteCfgRepo.Upsert(ctx, row); err != nil {
				return err
			}
			log.Println("Created blank features site_config")
			if s.installedThemeRepo != nil {
				if err := s.installedThemeRepo.SetActive(ctx, builtinthemes.BlankSiteDefaultThemeID); err != nil {
					log.Printf("Warning: could not activate %s theme: %v", builtinthemes.BlankSiteDefaultThemeID, err)
				} else {
					log.Printf("Activated %s theme for blank site", builtinthemes.BlankSiteDefaultThemeID)
					if s.themePageService != nil {
						if err := s.themePageService.SeedThemePages(ctx, builtinthemes.BlankSiteDefaultThemeID); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	log.Println("Blank-site seed completed")
	return nil
}

func blankGlobalConfig() model.JSONMap {
	return model.JSONMap{
		"identity": model.JSONMap{
			"name":          model.JSONMap{"zh": "My Site"},
			"localeMode":    "mono-zh",
			"defaultLocale": "zh",
		},
		"brand":  model.JSONMap{"logo": model.JSONMap{"light": ""}, "favicon": "", "ogImage": "", "primaryColor": "#1e40af"},
		"author": model.JSONMap{"name": "", "socials": []any{}},
		"footer": model.JSONMap{},
		"seo":    model.JSONMap{},
	}
}

func blankFeaturesConfig() model.JSONMap {
	return model.JSONMap{
		"publicPages": model.JSONMap{
			"home":         true,
			"blog":         true,
			"contact":      true,
			"about":        false,
			"experts":      false,
			"coreServices": false,
			"advantages":   false,
			"cases":        false,
		},
		"blog": model.JSONMap{
			"comments": true,
			"rss":      true,
		},
	}
}

// getInitialConfig returns an initial empty config structure for a page
func getInitialConfig(pageKey model.PageKey) model.JSONMap {
	// Return minimal valid config structure
	// These are placeholders that can be edited via the admin UI
	switch pageKey {
	case model.PageKeyHome:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "欢迎来到北辰工作室",
					"en": "Welcome to Northstar Studio",
				},
				"subtitle": model.JSONMap{
					"zh": "专业的咨询服务",
					"en": "Professional Consulting Services",
				},
				"backgroundImage": model.JSONMap{
					"url": "/images/hero-bg.jpg",
					"alt": model.JSONMap{
						"zh": "首页背景",
						"en": "Home Background",
					},
				},
			},
			"about": model.JSONMap{
				"title": model.JSONMap{
					"zh": "关于我们",
					"en": "About Us",
				},
				"descriptions": []interface{}{
					model.JSONMap{
						"zh": "北辰工作室提供专业的咨询服务",
						"en": "Northstar Studio provides professional consulting services",
					},
				},
				"image": model.JSONMap{
					"url": "/images/about.jpg",
					"alt": model.JSONMap{
						"zh": "关于我们",
						"en": "About Us",
					},
				},
				"cta": model.JSONMap{
					"label": model.JSONMap{
						"zh": "了解更多",
						"en": "Learn More",
					},
					"href":   "/about",
					"target": "_self",
				},
			},
			"advantages": model.JSONMap{
				"title": model.JSONMap{
					"zh": "我们的优势",
					"en": "Our Advantages",
				},
				"cards": []interface{}{},
			},
			"coreServices": model.JSONMap{
				"title": model.JSONMap{
					"zh": "核心服务",
					"en": "Core Services",
				},
				"items": []interface{}{},
			},
		}
	case model.PageKeyAbout:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "关于我们",
					"en": "About Us",
				},
				"backgroundImage": model.JSONMap{
					"url": "/images/about-hero.jpg",
					"alt": model.JSONMap{
						"zh": "关于我们背景",
						"en": "About Us Background",
					},
				},
			},
			"companyProfile": model.JSONMap{
				"title": model.JSONMap{
					"zh": "公司简介",
					"en": "Company Profile",
				},
				"content": model.JSONMap{
					"zh": "北辰工作室成立于2020年",
					"en": "Northstar Studio was established in 2020",
				},
			},
			"blocks": []interface{}{},
		}
	case model.PageKeyAdvantages:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "我们的优势",
					"en": "Our Advantages",
				},
			},
			"blocks": []interface{}{},
		}
	case model.PageKeyCoreServices:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "核心服务",
					"en": "Core Services",
				},
			},
			"services": []interface{}{},
		}
	case model.PageKeyCases:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "案例展示",
					"en": "Case Studies",
				},
			},
			"cases": []interface{}{},
		}
	case model.PageKeyExperts:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "专家团队",
					"en": "Our Experts",
				},
			},
			"sectionTitle": model.JSONMap{
				"zh": "团队介绍",
				"en": "Team Introduction",
			},
			"experts": []interface{}{},
		}
	case model.PageKeyContact:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "联系我们",
					"en": "Contact Us",
				},
			},
			"contactInfo": model.JSONMap{
				"email": "hello@example.com",
				"phone": "+86 123 4567 8900",
			},
		}
	case model.PageKeyGlobal:
		return model.JSONMap{
			"header": model.JSONMap{
				"logo": model.JSONMap{
					"url": "/images/logo.svg",
					"alt": model.JSONMap{
						"zh": "北辰工作室",
						"en": "Northstar Studio",
					},
				},
				"navLinks": []interface{}{},
			},
			"footer": model.JSONMap{
				"links": []interface{}{},
			},
		}
	case model.PageKeyTheme:
		return model.JSONMap{
			"activeTheme": builtinthemes.DefaultFallbackThemeID,
		}
	default:
		return model.JSONMap{}
	}
}
