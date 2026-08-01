package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicBootstrapIncludesPublishedUnifiedPageFacts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.ContentDocument{},
		&model.InstalledTheme{},
		&model.Page{},
		&model.UnifiedPage{},
		&model.SiteConfig{},
	))

	unifiedRepo := repository.NewGormUnifiedPageRepository(db)
	require.NoError(t, unifiedRepo.Create(t.Context(), &model.UnifiedPage{
		Slug:             "launch",
		ZhTitle:          "发布页",
		EnTitle:          "Launch",
		Mode:             model.PageModeComposable,
		Status:           "published",
		PublishedConfig:  model.NullableJSONMap{"sections": []any{}},
		PublishedVersion: 2,
		ShowInNav:        true,
		SortOrder:        8,
	}))
	require.NoError(t, unifiedRepo.Create(t.Context(), &model.UnifiedPage{
		Slug:   "draft-only",
		Mode:   model.PageModeComposable,
		Status: "draft",
	}))

	publicCache := cache.New(time.Minute)
	defer publicCache.Stop()
	handler := NewHandler(
		repository.NewGormContentDocumentRepository(db),
		repository.NewGormInstalledThemeRepository(db),
		repository.NewGormPageRepository(db),
		unifiedRepo,
		repository.NewGormSiteConfigRepository(db),
		publicCache,
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public/bootstrap", handler.PublicBootstrap)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/public/bootstrap?locale=zh", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		UnifiedPages []map[string]any `json:"unifiedPages"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.UnifiedPages, 1)
	require.Equal(t, "launch", payload.UnifiedPages[0]["slug"])
	require.Equal(t, true, payload.UnifiedPages[0]["showInNav"])
	require.Equal(t, float64(2), payload.UnifiedPages[0]["publishedVersion"])
	require.NotContains(t, payload.UnifiedPages[0], "publishedConfig")
}

func TestPublicBootstrapPageContentPrefersUnifiedPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.ContentDocument{},
		&model.InstalledTheme{},
		&model.Page{},
		&model.UnifiedPage{},
		&model.SiteConfig{},
	))

	unifiedRepo := repository.NewGormUnifiedPageRepository(db)
	require.NoError(t, unifiedRepo.Create(t.Context(), &model.UnifiedPage{
		Slug:             "home",
		Mode:             model.PageModeTemplate,
		TemplateKey:      "product-first/home@1",
		Status:           "published",
		PublishedConfig:  model.NullableJSONMap{"hero": map[string]any{"title": map[string]any{"zh": "从 Page"}}},
		PublishedVersion: 3,
	}))
	docRepo := repository.NewGormContentDocumentRepository(db)
	require.NoError(t, docRepo.Create(t.Context(), &model.ContentDocument{
		PageKey:          model.PageKeyHome,
		PublishedConfig:  model.JSONMap{"hero": map[string]any{"title": map[string]any{"zh": "从 content_doc"}}},
		PublishedVersion: 1,
	}))

	publicCache := cache.New(time.Minute)
	defer publicCache.Stop()
	handler := NewHandler(
		docRepo,
		repository.NewGormInstalledThemeRepository(db),
		repository.NewGormPageRepository(db),
		unifiedRepo,
		repository.NewGormSiteConfigRepository(db),
		publicCache,
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public/bootstrap", handler.PublicBootstrap)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/public/bootstrap?locale=zh&pageKey=home", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	pc, ok := payload["pageContent"].(map[string]any)
	require.True(t, ok, "pageContent missing: %v", payload)
	require.Equal(t, "page", pc["source"])
	require.Equal(t, "product-first/home@1", pc["templateKey"])
	cfg, _ := pc["config"].(map[string]any)
	hero, _ := cfg["hero"].(map[string]any)
	title, _ := hero["title"].(map[string]any)
	require.Equal(t, "从 Page", title["zh"])
}
