package db

import (
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
	}
}
