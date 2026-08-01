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

func TestContentToPageMigrator_MigrateHome(t *testing.T) {
	dsn := "file:ctp-" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UnifiedPage{}, &model.ContentDocument{}, &model.InstalledTheme{}))

	u := repository.NewGormUnifiedPageRepository(db)
	c := repository.NewGormContentDocumentRepository(db)
	th := repository.NewGormInstalledThemeRepository(db)
	require.NoError(t, th.Create(context.Background(), &model.InstalledTheme{
		ThemeID: builtinthemes.ProductFirst, Name: "PF", Source: "built-in", IsActive: true,
	}))
	require.NoError(t, c.Create(context.Background(), &model.ContentDocument{
		PageKey:         model.PageKeyHome,
		PublishedConfig: model.JSONMap{"hero": map[string]any{"title": map[string]any{"zh": "H"}}},
		DraftConfig:     model.JSONMap{"hero": map[string]any{"title": map[string]any{"zh": "H"}}},
		DraftVersion:    1,
		PublishedVersion: 1,
	}))

	m := NewContentToPageMigrator(u, c, th)
	res := m.MigrateHome(context.Background(), "", false)
	assert.Contains(t, []string{"created", "updated", "skipped"}, res.Action)
	assert.NotZero(t, res.PageID)

	page, err := u.FindBySlug(context.Background(), "home")
	require.NoError(t, err)
	assert.Equal(t, "product-first/home", page.TemplateKey)

	// force sync
	require.NoError(t, c.Update(context.Background(), &model.ContentDocument{
		PageKey:         model.PageKeyHome,
		DraftConfig:     model.JSONMap{"hero": map[string]any{"title": map[string]any{"zh": "NEW"}}},
		PublishedConfig: model.JSONMap{"hero": map[string]any{"title": map[string]any{"zh": "NEW"}}},
		DraftVersion:    2,
		PublishedVersion: 2,
	}))
	// Update may not work without full model - use UpdateDraft
	_, _ = c.UpdateDraft(context.Background(), model.PageKeyHome, 1, model.JSONMap{
		"hero": map[string]any{"title": map[string]any{"zh": "NEW"}},
	})
	res2 := m.MigrateHome(context.Background(), builtinthemes.ProductFirst, true)
	assert.Equal(t, "updated", res2.Action)
}

func TestSyncContentDocumentFromPage(t *testing.T) {
	dsn := "file:sync-" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ContentDocument{}, &model.UnifiedPage{}))
	c := repository.NewGormContentDocumentRepository(db)
	page := &model.UnifiedPage{
		Slug:             "home",
		DraftConfig:      model.JSONMap{"a": 1},
		DraftVersion:     3,
		PublishedConfig:  model.NullableJSONMap{"a": 1},
		PublishedVersion: 2,
	}
	require.NoError(t, SyncContentDocumentFromPage(context.Background(), c, page, model.PageKeyHome))
	doc, err := c.FindByPageKey(context.Background(), model.PageKeyHome)
	require.NoError(t, err)
	assert.Equal(t, float64(1), doc.DraftConfig["a"]) // json numbers may be float
	// re-fetch - map may keep int depending on driver
	assert.NotNil(t, doc.DraftConfig["a"])
	assert.GreaterOrEqual(t, doc.DraftVersion, 3)
}
