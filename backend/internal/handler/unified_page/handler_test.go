package unified_page

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidatePublicPageSlug(t *testing.T) {
	require.NoError(t, validatePublicPageSlug("launch-page"))
	require.Error(t, validatePublicPageSlug("Launch Page"))
	require.Error(t, validatePublicPageSlug("blog"))
	require.Error(t, validatePublicPageSlug("admin"))
	require.Error(t, validatePublicPageSlug("health"))
	require.Error(t, validatePublicPageSlug("version"))
	require.Error(t, validatePublicPageSlug("metrics"))
}

func TestAdminUpdatePersistsNavigationMetadataAndInvalidatesPublicCaches(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UnifiedPage{}, &model.PageVersion{}))

	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	page := &model.UnifiedPage{
		Slug:             "old-route",
		ZhTitle:          "旧标题",
		EnTitle:          "Old title",
		Mode:             model.PageModeComposable,
		Status:           "published",
		PublishedConfig:  model.NullableJSONMap{"sections": []any{}},
		PublishedVersion: 1,
	}
	require.NoError(t, pageRepo.Create(t.Context(), page))

	publicCache := cache.New(time.Minute)
	defer publicCache.Stop()
	publicCache.Set("bootstrap:zh:", "stale")
	publicCache.Set("pages:list:zh", "stale")
	publicCache.Set("page:old-route:zh", "stale")

	handler := NewHandler(
		pageRepo,
		versionRepo,
		service.NewUnifiedPageService(pageRepo, versionRepo),
		publicCache,
		nil,
	)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/admin/pages/:id", handler.AdminUpdate)

	body, err := json.Marshal(map[string]any{
		"slug":      "new-route",
		"zhTitle":   "新标题",
		"enTitle":   "New title",
		"showInNav": true,
		"sortOrder": 5,
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/pages/1", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	updated, err := pageRepo.FindByID(t.Context(), page.ID)
	require.NoError(t, err)
	require.Equal(t, "new-route", updated.Slug)
	require.Equal(t, "新标题", updated.ZhTitle)
	require.True(t, updated.ShowInNav)
	require.Equal(t, 5, updated.SortOrder)

	_, bootstrapCached := publicCache.Get("bootstrap:zh:")
	_, listCached := publicCache.Get("pages:list:zh")
	_, oldPageCached := publicCache.Get("page:old-route:zh")
	require.False(t, bootstrapCached)
	require.False(t, listCached)
	require.False(t, oldPageCached)
}

func TestPublicListExcludesPublishedRowsWithoutContent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UnifiedPage{}, &model.PageVersion{}))

	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	require.NoError(t, pageRepo.Create(t.Context(), &model.UnifiedPage{
		Slug:            "with-content",
		Mode:            model.PageModeComposable,
		Status:          "published",
		PublishedConfig: model.NullableJSONMap{"sections": []any{}},
	}))
	require.NoError(t, pageRepo.Create(t.Context(), &model.UnifiedPage{
		Slug:   "without-content",
		Mode:   model.PageModeComposable,
		Status: "published",
	}))

	publicCache := cache.New(time.Minute)
	defer publicCache.Stop()
	handler := NewHandler(
		pageRepo,
		versionRepo,
		service.NewUnifiedPageService(pageRepo, versionRepo),
		publicCache,
		nil,
	)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public/pages", handler.PublicList)

	recorder := performJSONRequest(t, router, http.MethodGet, "/public/pages?locale=zh", nil, nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Items, 1)
	require.Equal(t, "with-content", payload.Items[0]["slug"])
}

func TestAdminUpdateValidatesAndClearsParent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UnifiedPage{}, &model.PageVersion{}))

	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	parent := &model.UnifiedPage{Slug: "parent", Mode: model.PageModeComposable}
	require.NoError(t, pageRepo.Create(t.Context(), parent))
	child := &model.UnifiedPage{
		Slug:     "child",
		Mode:     model.PageModeComposable,
		ParentID: &parent.ID,
	}
	require.NoError(t, pageRepo.Create(t.Context(), child))

	publicCache := cache.New(time.Minute)
	defer publicCache.Stop()
	handler := NewHandler(
		pageRepo,
		versionRepo,
		service.NewUnifiedPageService(pageRepo, versionRepo),
		publicCache,
		nil,
	)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/admin/pages/:id", handler.AdminUpdate)

	cycle := performJSONRequest(
		t,
		router,
		http.MethodPut,
		"/admin/pages/1",
		map[string]any{"slug": "parent", "parentId": child.ID},
		nil,
	)
	require.Equal(t, http.StatusBadRequest, cycle.Code, cycle.Body.String())

	zeroParent := performJSONRequest(
		t,
		router,
		http.MethodPut,
		"/admin/pages/2",
		map[string]any{"slug": "child", "parentId": 0},
		nil,
	)
	require.Equal(t, http.StatusBadRequest, zeroParent.Code, zeroParent.Body.String())

	clearParent := performJSONRequest(
		t,
		router,
		http.MethodPut,
		"/admin/pages/2",
		map[string]any{"slug": "child", "parentId": nil},
		nil,
	)
	require.Equal(t, http.StatusOK, clearParent.Code, clearParent.Body.String())
	updatedChild, err := pageRepo.FindByID(t.Context(), child.ID)
	require.NoError(t, err)
	require.Nil(t, updatedChild.ParentID)
}
