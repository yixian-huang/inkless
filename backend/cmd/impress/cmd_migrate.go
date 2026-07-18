package main

import (
	"encoding/json"
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
		&model.Plugin{},
		&model.PluginSetting{},
	}
}

func openDatabase(dsn string) (*db.DB, error) {
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		dsn = "file:./data/impress.db?cache=shared&mode=rwc"
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

func openReadOnlyDatabase(dsn string) (*db.DB, error) {
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		dsn = "file:./data/impress.db?cache=shared&mode=rwc"
	}

	maxOpen := 1
	maxIdle := 1
	var maxLife time.Duration
	if db.IsPostgresDSN(dsn) {
		maxOpen = 25
		maxIdle = 5
		maxLife = 5 * time.Minute
	}

	return db.InitReadOnly(db.InitOptions{
		DSN:         dsn,
		MaxOpenConn: maxOpen,
		MaxIdleConn: maxIdle,
		MaxLifetime: maxLife,
		LogLevel:    logger.Warn,
	})
}

func migrateCmd() *cobra.Command {
	var dsn string
	var legacyStatusJSON bool

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
				"installed_themes", "form_submissions", "plugins", "plugin_settings",
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

	legacyStatusCmd := &cobra.Command{
		Use:   "legacy-site-status",
		Short: "Inspect retired multi-site schema data without modifying it",
		Long:  "Read-only preflight that counts legacy sites, site_users, and non-NULL RBAC site_id values. It never migrates or drops schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openReadOnlyDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer database.Close()

			status, err := db.InspectLegacyMultiSiteSchema(cmd.Context(), database)
			if err != nil {
				return fmt.Errorf("inspect legacy multi-site schema: %w", err)
			}

			if legacyStatusJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(status)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "sites.exists=%t sites.rows=%d\n", status.Sites.Exists, status.Sites.Rows)
			fmt.Fprintf(out, "site_users.exists=%t site_users.rows=%d\n", status.SiteUsers.Exists, status.SiteUsers.Rows)
			fmt.Fprintf(out, "roles.site_id.exists=%t roles.site_id.non_null_rows=%d\n", status.RoleSiteID.Exists, status.RoleSiteID.NonNullRows)
			fmt.Fprintf(out, "user_roles.site_id.exists=%t user_roles.site_id.non_null_rows=%d\n", status.UserRoleSiteID.Exists, status.UserRoleSiteID.NonNullRows)
			fmt.Fprintf(out, "hasLegacyData=%t\n", status.HasLegacyData)
			if status.HasLegacyData {
				fmt.Fprintf(out, "WARNING: %s\n", status.Recommendation)
			} else {
				fmt.Fprintln(out, status.Recommendation)
			}
			return nil
		},
	}
	legacyStatusCmd.Flags().BoolVar(&legacyStatusJSON, "json", false, "print the report as JSON")
	cmd.AddCommand(legacyStatusCmd)

	return cmd
}
