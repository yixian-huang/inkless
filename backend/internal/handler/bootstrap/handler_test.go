package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"

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
