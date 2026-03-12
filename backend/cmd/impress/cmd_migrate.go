package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/model"
)

// allModels returns the list of GORM models used for AutoMigrate,
// matching the list in cmd/server/main.go.
func allModels() []interface{} {
	return []interface{}{
		&model.User{},
		&model.RefreshToken{},
		&model.ContentDocument{},
		&model.ContentVersion{},
		&model.Media{},
		&model.PageView{},
		&model.Category{},
		&model.Tag{},
		&model.Article{},
		&model.BackupRecord{},
		&model.AuditEvent{},
		&model.Page{},
		&model.InstalledTheme{},
		&model.FormSubmission{},
	}
}

func openDatabase(dsn string) (*db.DB, error) {
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		dsn = "file:./data/blotting.db?cache=shared&mode=rwc"
	}

	maxOpen := 1
	maxIdle := 1
	var maxLife time.Duration
	if db.IsPostgresDSN(dsn) {
		maxOpen = 25
		maxIdle = 5
		maxLife = 5 * time.Minute
	}

	return db.Init(db.InitOptions{
		DSN:         dsn,
		MaxOpenConn: maxOpen,
		MaxIdleConn: maxIdle,
		MaxLifetime: maxLife,
		LogLevel:    logger.Warn,
	})
}

func migrateCmd() *cobra.Command {
	var dsn string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration management",
		Long:  "Run or check status of GORM AutoMigrate database migrations.",
	}

	cmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Database DSN (default: DB_DSN env var or SQLite)")

	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations (GORM AutoMigrate)",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			migrator := db.NewMigrator(database)
			if err := migrator.AutoMigrate(allModels()...); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			fmt.Println("Migrations applied successfully.")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Drop all managed tables (destructive!)",
		Long:  "Drop all tables managed by GORM AutoMigrate. USE WITH CAUTION.",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			migrator := db.NewMigrator(database)
			if err := migrator.DropTable(allModels()...); err != nil {
				return fmt.Errorf("drop tables failed: %w", err)
			}

			fmt.Println("All managed tables dropped.")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show which tables exist",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			migrator := db.NewMigrator(database)
			models := allModels()
			names := []string{
				"users", "refresh_tokens", "content_documents", "content_versions",
				"media", "page_views", "categories", "tags",
				"articles", "backup_records", "audit_events", "pages",
				"installed_themes", "form_submissions",
			}

			fmt.Println("Table Status:")
			for i, m := range models {
				exists := migrator.HasTable(m)
				status := "missing"
				if exists {
					status = "ok"
				}
				fmt.Printf("  %-24s %s\n", names[i], status)
			}
			return nil
		},
	})

	return cmd
}
