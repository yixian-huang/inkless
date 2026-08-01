package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

func setupHomeMigrate(t *testing.T) (*HomePageMigrator, repository.UnifiedPageRepository, repository.ContentDocumentRepository) {
	t.Helper()
	// Unique DSN per test — shared in-memory SQLite cross-pollutes.
	dsn := "file:home-migrate-" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UnifiedPage{}, &model.ContentDocument{}))
	u := repository.NewGormUnifiedPageRepository(db)
	c := repository.NewGormContentDocumentRepository(db)
	return NewHomePageMigrator(u, c), u, c
}

func TestEnsureHomePage_CreatesFromContentDocuments(t *testing.T) {
	m, u, c := setupHomeMigrate(t)
	ctx := context.Background()
	require.NoError(t, c.Create(ctx, &model.ContentDocument{
		PageKey:          model.PageKeyHome,
		PublishedConfig:  model.JSONMap{"hero": map[string]any{"title": map[string]any{"zh": "产品"}}},
		PublishedVersion: 2,
		DraftVersion:     2,
	}))

	require.NoError(t, m.EnsureHomePage(ctx, builtinthemes.ProductFirst))
	page, err := u.FindBySlug(ctx, "home")
	require.NoError(t, err)
	assert.Equal(t, "product-first/home", page.TemplateKey)
	assert.Equal(t, model.PageModeTemplate, page.Mode)
	assert.Equal(t, "published", page.Status)
	assert.NotEmpty(t, page.PublishedConfig["hero"])
}

func TestEnsureHomePage_IdempotentNoOverwrite(t *testing.T) {
	m, u, c := setupHomeMigrate(t)
	ctx := context.Background()
	require.NoError(t, u.Create(ctx, &model.UnifiedPage{
		Slug:             "home",
		Mode:             model.PageModeTemplate,
		TemplateKey:      "product-first/home",
		Status:           "published",
		PublishedVersion: 3,
		PublishedConfig:  model.NullableJSONMap{"hero": map[string]any{"title": "user"}},
		DraftConfig:      model.JSONMap{"hero": map[string]any{"title": "user"}},
	}))
	require.NoError(t, c.Create(ctx, &model.ContentDocument{
		PageKey:         model.PageKeyHome,
		PublishedConfig: model.JSONMap{"hero": map[string]any{"title": "legacy"}},
	}))

	require.NoError(t, m.EnsureHomePage(ctx, builtinthemes.ProductFirst))
	page, err := u.FindBySlug(ctx, "home")
	require.NoError(t, err)
	assert.Equal(t, "user", page.PublishedConfig["hero"].(map[string]any)["title"])
	assert.Equal(t, 3, page.PublishedVersion)
}

func TestTemplateKeyForTheme(t *testing.T) {
	assert.Equal(t, "product-first/home", TemplateKeyForTheme(builtinthemes.ProductFirst))
	assert.Equal(t, "blog-first/home", TemplateKeyForTheme(builtinthemes.BlogFirst))
}
