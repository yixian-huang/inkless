package seed

import (
	"context"
	"testing"

	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSeeder_SeedUsers(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	userRepo := repository.NewGormUserRepository(db)

	ctx := context.Background()

	// First seed should create users
	err := seeder.SeedUsers(ctx)
	require.NoError(t, err)

	// Verify admin user was created
	adminUser, err := userRepo.FindByUsername(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", adminUser.Username)
	assert.Equal(t, model.RoleAdmin, adminUser.Role)
	assert.NotEmpty(t, adminUser.PasswordHash)

	// Verify password can be verified
	err = auth.VerifyPassword(adminUser.PasswordHash, "admin123")
	assert.NoError(t, err)

	// Verify editor user was created
	editorUser, err := userRepo.FindByUsername(ctx, "editor")
	require.NoError(t, err)
	assert.Equal(t, "editor", editorUser.Username)
	assert.Equal(t, model.RoleEditor, editorUser.Role)
	assert.NotEmpty(t, editorUser.PasswordHash)

	// Verify password can be verified
	err = auth.VerifyPassword(editorUser.PasswordHash, "editor123")
	assert.NoError(t, err)
}

func TestSeeder_SeedUsers_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	userRepo := repository.NewGormUserRepository(db)

	ctx := context.Background()

	// First seed
	err := seeder.SeedUsers(ctx)
	require.NoError(t, err)

	// Get initial admin user
	initialAdminUser, err := userRepo.FindByUsername(ctx, "admin")
	require.NoError(t, err)
	initialAdminID := initialAdminUser.ID

	// Get initial user count
	_, count, err := userRepo.List(ctx, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Second seed should be idempotent
	err = seeder.SeedUsers(ctx)
	require.NoError(t, err)

	// Verify user count unchanged
	_, count, err = userRepo.List(ctx, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify admin user ID unchanged (not recreated)
	adminUser, err := userRepo.FindByUsername(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, initialAdminID, adminUser.ID)
}

func TestSeeder_SeedContentDocuments(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)

	ctx := context.Background()

	// Seed content documents
	err := seeder.SeedContentDocuments(ctx)
	require.NoError(t, err)

	// Verify all page keys have documents
	for _, pageKey := range model.ValidPageKeys {
		doc, err := contentRepo.FindByPageKey(ctx, pageKey)
		require.NoError(t, err, "Document for %s should exist", pageKey)
		assert.Equal(t, pageKey, doc.PageKey)
		assert.Equal(t, 1, doc.DraftVersion)
		assert.Equal(t, 1, doc.PublishedVersion)
		assert.NotNil(t, doc.DraftConfig)
		assert.NotNil(t, doc.PublishedConfig)
		assert.NotEmpty(t, doc.DraftConfig)
		assert.NotEmpty(t, doc.PublishedConfig)
	}
}

func TestSeeder_SeedContentDocuments_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)

	ctx := context.Background()

	// First seed
	err := seeder.SeedContentDocuments(ctx)
	require.NoError(t, err)

	// Get initial home document
	homeDoc1, err := contentRepo.FindByPageKey(ctx, model.PageKeyHome)
	require.NoError(t, err)

	// Second seed should be idempotent
	err = seeder.SeedContentDocuments(ctx)
	require.NoError(t, err)

	// Verify document unchanged
	homeDoc2, err := contentRepo.FindByPageKey(ctx, model.PageKeyHome)
	require.NoError(t, err)
	assert.Equal(t, homeDoc1.PageKey, homeDoc2.PageKey)
	assert.Equal(t, homeDoc1.DraftVersion, homeDoc2.DraftVersion)
	assert.Equal(t, homeDoc1.PublishedVersion, homeDoc2.PublishedVersion)
}

func TestSeeder_SeedAll(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	userRepo := repository.NewGormUserRepository(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)

	ctx := context.Background()

	// Seed all data
	err := seeder.SeedAll(ctx)
	require.NoError(t, err)

	// Verify users were created
	adminUser, err := userRepo.FindByUsername(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", adminUser.Username)

	editorUser, err := userRepo.FindByUsername(ctx, "editor")
	require.NoError(t, err)
	assert.Equal(t, "editor", editorUser.Username)

	// Verify all content documents were created
	docs, err := contentRepo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, len(model.ValidPageKeys))
}

func TestSeeder_SeedAll_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	userRepo := repository.NewGormUserRepository(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)

	ctx := context.Background()

	// Seed twice
	err := seeder.SeedAll(ctx)
	require.NoError(t, err)

	err = seeder.SeedAll(ctx)
	require.NoError(t, err)

	// Verify correct counts
	users, count, err := userRepo.List(ctx, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, users, 2)

	docs, err := contentRepo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, len(model.ValidPageKeys))
}

func TestGetInitialConfig_Home(t *testing.T) {
	config := getInitialConfig(model.PageKeyHome)
	assert.NotNil(t, config)

	// Verify required sections exist
	assert.Contains(t, config, "hero")
	assert.Contains(t, config, "about")
	assert.Contains(t, config, "advantages")
	assert.Contains(t, config, "coreServices")

	// Verify hero structure
	hero := config["hero"].(model.JSONMap)
	assert.Contains(t, hero, "title")
	assert.Contains(t, hero, "subtitle")
	assert.Contains(t, hero, "backgroundImage")

	// Verify bilingual fields
	title := hero["title"].(model.JSONMap)
	assert.Contains(t, title, "zh")
	assert.Contains(t, title, "en")
	assert.NotEmpty(t, title["zh"])
	assert.NotEmpty(t, title["en"])
}

func TestGetInitialConfig_About(t *testing.T) {
	config := getInitialConfig(model.PageKeyAbout)
	assert.NotNil(t, config)
	assert.Contains(t, config, "hero")
	assert.Contains(t, config, "companyProfile")
	assert.Contains(t, config, "blocks")
}

func TestGetInitialConfig_AllPageKeys(t *testing.T) {
	for _, pageKey := range model.ValidPageKeys {
		t.Run(string(pageKey), func(t *testing.T) {
			config := getInitialConfig(pageKey)
			assert.NotNil(t, config)
			assert.NotEmpty(t, config, "Config for %s should not be empty", pageKey)
		})
	}
}

func TestSeeder_SeedContentDocuments_InitialConfigHasBothDraftAndPublished(t *testing.T) {
	db := setupTestDB(t)
	seeder := newTestSeeder(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)

	ctx := context.Background()

	err := seeder.SeedContentDocuments(ctx)
	require.NoError(t, err)

	// Verify both draft and published configs are identical initially
	homeDoc, err := contentRepo.FindByPageKey(ctx, model.PageKeyHome)
	require.NoError(t, err)

	assert.NotNil(t, homeDoc.DraftConfig)
	assert.NotNil(t, homeDoc.PublishedConfig)
	assert.NotEmpty(t, homeDoc.DraftConfig)
	assert.NotEmpty(t, homeDoc.PublishedConfig)

	// Both should have version 1
	assert.Equal(t, 1, homeDoc.DraftVersion)
	assert.Equal(t, 1, homeDoc.PublishedVersion)
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	opts := db.InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	}

	database, err := db.Init(opts)
	require.NoError(t, err)

	// Enable foreign keys for SQLite
	database.Exec("PRAGMA foreign_keys = ON")

	// Run migrations
	migrator := db.NewMigrator(database)
	err = migrator.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.ContentDocument{},
		&model.ContentVersion{},
		&model.InstalledTheme{},
		&model.Page{},
		&model.UnifiedPage{},
		&model.PageTemplate{},
	)
	require.NoError(t, err)

	return database.DB
}

func newTestSeeder(db *gorm.DB) *Seeder {
	userRepo := repository.NewGormUserRepository(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)
	themeRepo := repository.NewGormInstalledThemeRepository(db)
	pageRepo := repository.NewGormPageRepository(db)
	themePageSvc := service.NewThemePageService(pageRepo)
	unifiedPageRepo := repository.NewGormUnifiedPageRepository(db)
	templateRepo := repository.NewGormPageTemplateRepository(db)
	return NewSeeder(userRepo, contentRepo, themeRepo, themePageSvc, unifiedPageRepo, templateRepo, nil)
}

func TestBlankSiteSeed_CreatesGlobalWithDefaults(t *testing.T) {
	db := setupTestDB(t)

	// Also migrate SiteConfig table for this test
	err := db.AutoMigrate(&model.SiteConfig{})
	require.NoError(t, err)

	userRepo := repository.NewGormUserRepository(db)
	contentRepo := repository.NewGormContentDocumentRepository(db)
	installedThemeRepo := repository.NewGormInstalledThemeRepository(db)
	pageRepo := repository.NewGormPageRepository(db)
	themePageService := service.NewThemePageService(pageRepo)
	unifiedPageRepo := repository.NewGormUnifiedPageRepository(db)
	pageTemplateRepo := repository.NewGormPageTemplateRepository(db)
	siteCfgRepo := repository.NewGormSiteConfigRepository(db)

	s := NewSeeder(userRepo, contentRepo, installedThemeRepo, themePageService, unifiedPageRepo, pageTemplateRepo, siteCfgRepo)

	if err := s.BlankSiteSeed(context.Background()); err != nil {
		t.Fatalf("blank seed: %v", err)
	}
	doc, err := contentRepo.FindByPageKey(context.Background(), model.PageKeyGlobal)
	if err != nil {
		t.Fatalf("find global: %v", err)
	}
	identity, ok := doc.PublishedConfig["identity"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected identity map in published config, got: %#v", doc.PublishedConfig["identity"])
	}
	if identity["localeMode"] != "mono-zh" {
		t.Fatalf("expected mono-zh, got: %v", identity["localeMode"])
	}
}
