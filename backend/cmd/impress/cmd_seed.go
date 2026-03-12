package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/seed"
	"blotting-consultancy/internal/service"
)

func seedCmd() *cobra.Command {
	var dsn string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long:  "Populate the database with default admin user, example content, and themes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := openDatabase(dsn)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			userRepo := repository.NewGormUserRepository(database.DB)
			contentDocRepo := repository.NewGormContentDocumentRepository(database.DB)
			installedThemeRepo := repository.NewGormInstalledThemeRepository(database.DB)
			pageRepo := repository.NewGormPageRepository(database.DB)
			themePageService := service.NewThemePageService(pageRepo)

			seeder := seed.NewSeeder(userRepo, contentDocRepo, installedThemeRepo, themePageService)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := seeder.SeedAll(ctx); err != nil {
				return fmt.Errorf("seeding failed: %w", err)
			}

			fmt.Println("Database seeded successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN (default: DB_DSN env var or SQLite)")
	return cmd
}
