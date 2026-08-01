package db

import (
	"time"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/model"

	"gorm.io/gorm"
)

// DataMigrations returns the list of data migrations to run after AutoMigrate
func DataMigrations() []Migration {
	return []Migration{
		{
			ID: "001_migrate_category_id_to_article_categories",
			Up: func(db *gorm.DB) error {
				// Copy existing category_id relationships to the article_categories join table
				// Only run if the join table is empty (first migration)
				var count int64
				if err := db.Table("article_categories").Count(&count).Error; err != nil {
					// Table might not exist yet, skip
					return nil
				}
				if count > 0 {
					return nil // Already has data, skip
				}

				return db.Exec(
					"INSERT INTO article_categories (article_id, category_id) SELECT id, category_id FROM articles WHERE category_id IS NOT NULL",
				).Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("DELETE FROM article_categories").Error
			},
		},
		{
			ID: "002_set_admin_super_admin",
			Up: func(db *gorm.DB) error {
				return db.Exec("UPDATE users SET is_super_admin = ? WHERE username = ?", true, "admin").Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("UPDATE users SET is_super_admin = ? WHERE username = ?", false, "admin").Error
			},
		},
		{
			ID: "003_create_ai_configs",
			Up: func(db *gorm.DB) error {
				return db.AutoMigrate(&model.AIConfig{})
			},
			Down: func(db *gorm.DB) error {
				return db.Migrator().DropTable(&model.AIConfig{})
			},
		},
		{
			ID: "004_preserve_legacy_meilisearch_index_prefix",
			Up: func(db *gorm.DB) error {
				if !db.Migrator().HasTable(&model.Plugin{}) {
					return nil
				}

				var plugins []model.Plugin
				if err := db.Where("plugin_id = ?", "mls-search").Find(&plugins).Error; err != nil {
					return err
				}
				for i := range plugins {
					settings := plugins[i].Settings
					if settings == nil {
						settings = make(model.JSONMap)
					}
					if _, configured := settings["index_prefix"]; configured {
						continue
					}
					settings["index_prefix"] = "impress_"
					if err := db.Model(&model.Plugin{}).
						Where("id = ?", plugins[i].ID).
						Update("settings", settings).Error; err != nil {
						return err
					}
				}
				return nil
			},
			// Removing the compatibility marker on rollback could make an existing
			// deployment silently switch to a different index namespace.
			Down: func(*gorm.DB) error { return nil },
		},
		{
			// theme-as-templates T1/T2: ensure slug=home Page from content_documents when active theme is product/blog-first
			// Kept in db package (no import of service) to avoid cycles.
			ID: "005_ensure_home_unified_page",
			Up: func(db *gorm.DB) error {
				if !db.Migrator().HasTable(&model.UnifiedPage{}) {
					return nil
				}
				_ = db.AutoMigrate(&model.UnifiedPage{})

				var active model.InstalledTheme
				if err := db.Where("is_active = ?", true).First(&active).Error; err != nil {
					return nil
				}
				var templateKey string
				switch active.ThemeID {
				case builtinthemes.ProductFirst:
					templateKey = "product-first/home"
				case builtinthemes.BlogFirst:
					templateKey = "blog-first/home"
				default:
					return nil
				}

				var existing model.UnifiedPage
				err := db.Where("slug = ?", "home").First(&existing).Error
				if err == nil {
					// Upgrade empty template binding only
					updates := map[string]interface{}{}
					if existing.TemplateKey == "" {
						updates["template_key"] = templateKey
					}
					if existing.Mode == "" || existing.Mode == model.PageModeComposable {
						updates["mode"] = model.PageModeTemplate
					}
					if len(updates) > 0 {
						return db.Model(&existing).Updates(updates).Error
					}
					return nil
				}

				cfg := model.JSONMap{}
				var doc model.ContentDocument
				if err := db.Where("page_key = ?", "home").First(&doc).Error; err == nil {
					if len(doc.PublishedConfig) > 0 {
						cfg = model.JSONMap(doc.PublishedConfig)
					} else if len(doc.DraftConfig) > 0 {
						cfg = model.JSONMap(doc.DraftConfig)
					}
				}
				if len(cfg) == 0 {
					cfg = model.JSONMap{"_templateKey": templateKey}
				}
				now := time.Now().UTC()
				page := model.UnifiedPage{
					Slug:             "home",
					ZhTitle:          "首页",
					EnTitle:          "Home",
					Mode:             model.PageModeTemplate,
					TemplateKey:      templateKey,
					DraftConfig:      cfg,
					DraftVersion:     1,
					PublishedConfig:  model.NullableJSONMap(cfg),
					PublishedVersion: 1,
					Status:           "published",
					ShowInNav:        true,
					PublishedAt:      &now,
				}
				return db.Create(&page).Error
			},
			Down: func(*gorm.DB) error { return nil },
		},
	}
}
