package seed

import (
	"context"
	"log"
	"strings"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/auth"
)

// Seeder handles idempotent seeding of initial data
type Seeder struct {
	userRepo           repository.UserRepository
	contentRepo        repository.ContentDocumentRepository
	installedThemeRepo repository.InstalledThemeRepository
	themePageService   *service.ThemePageService
}

// NewSeeder creates a new seeder instance
func NewSeeder(userRepo repository.UserRepository, contentRepo repository.ContentDocumentRepository, installedThemeRepo repository.InstalledThemeRepository, themePageService *service.ThemePageService) *Seeder {
	return &Seeder{
		userRepo:           userRepo,
		contentRepo:        contentRepo,
		installedThemeRepo: installedThemeRepo,
		themePageService:   themePageService,
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

// SeedInstalledThemes creates the default corporate-classic theme if it doesn't exist
func (s *Seeder) SeedInstalledThemes(ctx context.Context) error {
	existing, err := s.installedThemeRepo.FindByThemeID(ctx, "corporate-classic")
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}

	if existing != nil {
		log.Println("InstalledTheme corporate-classic already exists, skipping")
		return nil
	}

	theme := &model.InstalledTheme{
		ThemeID:     "corporate-classic",
		Name:        "Corporate Classic",
		NameZh:      "企业经典",
		Description: "专业企业官网，含首页、关于、优势、服务、案例、专家、联系",
		Author:      "Blotting Consultancy",
		Version:     "1.0.0",
		Source:      "built-in",
		IsActive:    true,
		Preview:     "linear-gradient(135deg, #1a5f8f 0%, #8bc34a 100%)",
	}

	if err := s.installedThemeRepo.Create(ctx, theme); err != nil {
		return err
	}

	log.Println("Created InstalledTheme: corporate-classic (active)")
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

// getInitialConfig returns an initial empty config structure for a page
func getInitialConfig(pageKey model.PageKey) model.JSONMap {
	// Return minimal valid config structure
	// These are placeholders that can be edited via the admin UI
	switch pageKey {
	case model.PageKeyHome:
		return model.JSONMap{
			"hero": model.JSONMap{
				"title": model.JSONMap{
					"zh": "欢迎来到印迹咨询",
					"en": "Welcome to Blotting Consultancy",
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
						"zh": "印迹咨询提供专业的咨询服务",
						"en": "Blotting Consultancy provides professional consulting services",
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
					"zh": "印迹咨询成立于2020年",
					"en": "Blotting Consultancy was established in 2020",
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
				"email": "info@blotting.com",
				"phone": "+86 123 4567 8900",
			},
		}
	case model.PageKeyGlobal:
		return model.JSONMap{
			"header": model.JSONMap{
				"logo": model.JSONMap{
					"url": "/images/logo.svg",
					"alt": model.JSONMap{
						"zh": "印迹咨询",
						"en": "Blotting Consultancy",
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
			"activeTheme": "corporate-classic",
		}
	default:
		return model.JSONMap{}
	}
}
