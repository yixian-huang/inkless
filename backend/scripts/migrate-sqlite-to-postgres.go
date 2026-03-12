package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

func main() {
	fmt.Println("=== SQLite to PostgreSQL Migration Tool ===")

	// Source: SQLite
	srcDSN := os.Getenv("SOURCE_DB_DSN")
	if srcDSN == "" {
		srcDSN = "data/blotting.db"
		fmt.Printf("Using default SOURCE_DB_DSN: %s\n", srcDSN)
	} else {
		fmt.Printf("Source database: %s\n", srcDSN)
	}

	// Target: PostgreSQL
	targetDSN := os.Getenv("TARGET_DB_DSN")
	if targetDSN == "" {
		log.Fatal("ERROR: TARGET_DB_DSN environment variable required\n" +
			"Example: export TARGET_DB_DSN=\"host=localhost user=blotting_user password=blotting_dev_password dbname=blotting_cms port=5432 sslmode=disable\"")
	}
	fmt.Printf("Target database: PostgreSQL\n\n")

	// Connect to source SQLite
	fmt.Println("Connecting to source database...")
	srcDB, err := db.Init(db.InitOptions{DSN: srcDSN})
	if err != nil {
		log.Fatalf("Failed to connect to source database: %v", err)
	}
	defer srcDB.Close()
	fmt.Println("✓ Connected to source database")

	// Connect to target PostgreSQL
	fmt.Println("Connecting to target database...")
	targetDB, err := db.Init(db.InitOptions{DSN: targetDSN})
	if err != nil {
		log.Fatalf("Failed to connect to target database: %v", err)
	}
	defer targetDB.Close()
	fmt.Println("✓ Connected to target database")

	// Verify target database is empty or prompt for confirmation
	var userCount, docCount, versionCount int64
	targetDB.Model(&model.User{}).Count(&userCount)
	targetDB.Model(&model.ContentDocument{}).Count(&docCount)
	targetDB.Model(&model.ContentVersion{}).Count(&versionCount)

	if userCount > 0 || docCount > 0 || versionCount > 0 {
		fmt.Printf("WARNING: Target database is not empty!\n")
		fmt.Printf("  Users: %d\n", userCount)
		fmt.Printf("  Content Documents: %d\n", docCount)
		fmt.Printf("  Content Versions: %d\n", versionCount)
		fmt.Printf("\nThis migration will attempt to insert records and may fail on duplicates.\n")
		fmt.Printf("To proceed, type 'yes': ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Migration cancelled.")
			return
		}
	}

	// Initialize repositories
	srcUserRepo := repository.NewGormUserRepository(srcDB.DB)
	targetUserRepo := repository.NewGormUserRepository(targetDB.DB)

	srcDocRepo := repository.NewGormContentDocumentRepository(srcDB.DB)
	targetDocRepo := repository.NewGormContentDocumentRepository(targetDB.DB)

	srcVersionRepo := repository.NewGormContentVersionRepository(srcDB.DB)
	targetVersionRepo := repository.NewGormContentVersionRepository(targetDB.DB)

	ctx := context.Background()

	// Statistics
	stats := struct {
		UsersOK, UsersFailed         int
		DocsOK, DocsFailed           int
		VersionsOK, VersionsFailed   int
	}{}

	// Migrate Users
	fmt.Println("=== Migrating Users ===")
	users, _, err := srcUserRepo.List(ctx, 0, 10000)
	if err != nil {
		log.Fatalf("Failed to fetch users from source: %v", err)
	}
	fmt.Printf("Found %d users in source database\n", len(users))

	for _, user := range users {
		if err := targetUserRepo.Create(ctx, user); err != nil {
			log.Printf("  ✗ Failed to migrate user %s (ID: %d): %v", user.Username, user.ID, err)
			stats.UsersFailed++
		} else {
			fmt.Printf("  ✓ Migrated user: %s (ID: %d, Role: %s)\n", user.Username, user.ID, user.Role)
			stats.UsersOK++
		}
	}

	// Migrate Content Documents
	fmt.Println("\n=== Migrating Content Documents ===")
	for _, pageKey := range model.ValidPageKeys {
		doc, err := srcDocRepo.FindByPageKey(ctx, pageKey)
		if err != nil {
			// Document may not exist in source, skip silently
			continue
		}

		if err := targetDocRepo.Create(ctx, doc); err != nil {
			log.Printf("  ✗ Failed to migrate document %s: %v", pageKey, err)
			stats.DocsFailed++
		} else {
			fmt.Printf("  ✓ Migrated document: %s (draft v%d, published v%d)\n",
				pageKey, doc.DraftVersion, doc.PublishedVersion)
			stats.DocsOK++
		}
	}

	// Migrate Content Versions
	fmt.Println("\n=== Migrating Content Versions ===")
	for _, pageKey := range model.ValidPageKeys {
		versions, _, err := srcVersionRepo.ListByPageKey(ctx, pageKey, 0, 10000)
		if err != nil {
			// Page may not have versions, skip silently
			continue
		}

		if len(versions) == 0 {
			continue
		}

		fmt.Printf("Found %d versions for page: %s\n", len(versions), pageKey)
		for _, version := range versions {
			if err := targetVersionRepo.Create(ctx, version); err != nil {
				log.Printf("  ✗ Failed to migrate version %s v%d: %v", pageKey, version.Version, err)
				stats.VersionsFailed++
			} else {
				fmt.Printf("  ✓ Migrated version: %s v%d (published at %s)\n",
					pageKey, version.Version, version.PublishedAt.Format("2006-01-02 15:04:05"))
				stats.VersionsOK++
			}
		}
	}

	// Summary
	fmt.Println("\n=== Migration Summary ===")
	fmt.Printf("Users:             %d successful, %d failed\n", stats.UsersOK, stats.UsersFailed)
	fmt.Printf("Content Documents: %d successful, %d failed\n", stats.DocsOK, stats.DocsFailed)
	fmt.Printf("Content Versions:  %d successful, %d failed\n", stats.VersionsOK, stats.VersionsFailed)

	if stats.UsersFailed > 0 || stats.DocsFailed > 0 || stats.VersionsFailed > 0 {
		fmt.Println("\n⚠ Migration completed with errors. Review logs above.")
		os.Exit(1)
	}

	fmt.Println("\n✓ Migration completed successfully!")
	fmt.Println("\nNote: Refresh tokens were not migrated as they expire quickly.")
	fmt.Println("      Users will need to re-login after migration.")
}
