package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	publicHandler "github.com/yixian-huang/inkless/backend/internal/handler/public"
)

func setupContentRouter(t *testing.T) (*gin.Engine, *gorm.DB, *cache.Cache) {
	t.Helper()
	// Unique DSN per test — shared in-memory SQLite would cross-pollute parallel/package tests.
	dsn := fmt.Sprintf("file:content-admin-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ContentDocument{}, &model.ContentVersion{}, &model.User{}))

	docRepo := repository.NewGormContentDocumentRepository(db)
	verRepo := repository.NewGormContentVersionRepository(db)
	validationSvc := service.NewValidationService()
	contentSvc := service.NewContentService(db, docRepo, verRepo, validationSvc)
	publicCache := cache.New(time.Minute)
	h := NewHandler(db, docRepo, verRepo, validationSvc, contentSvc, nil, publicCache)
	pub := publicHandler.NewHandler(docRepo, nil, nil, publicCache)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.UserContextKey), &middleware.UserContext{
			UserID:   1,
			Username: "admin",
			Role:     model.RoleAdmin,
		})
		c.Next()
	})
	r.GET("/admin/content/:pageKey/draft", h.GetDraft)
	r.PUT("/admin/content/:pageKey/draft", h.UpdateDraft)
	r.POST("/admin/content/:pageKey/validate", h.Validate)
	r.POST("/admin/content/:pageKey/publish", h.Publish)
	r.GET("/public/content/:pageKey", pub.GetPublicContent)
	return r, db, publicCache
}

func productHomeConfig() map[string]interface{} {
	return map[string]interface{}{
		"hero": map[string]interface{}{
			"title":    map[string]interface{}{"zh": "产品", "en": "Product"},
			"subtitle": map[string]interface{}{"zh": "副标题", "en": "Subtitle"},
			"media": map[string]interface{}{
				"url":     "/uploads/hero.png",
				"alt":     "Hero",
				"caption": "Main shot",
			},
		},
		"showcase": map[string]interface{}{
			"title": map[string]interface{}{"zh": "展示", "en": "Showcase"},
			"items": []interface{}{
				map[string]interface{}{"url": "/a.png", "alt": "A", "caption": "one"},
			},
		},
		"features": map[string]interface{}{
			"title": map[string]interface{}{"zh": "能力", "en": "Features"},
			"items": []interface{}{
				map[string]interface{}{
					"title":       map[string]interface{}{"zh": "快", "en": "Fast"},
					"description": map[string]interface{}{"zh": "快描述", "en": "Fast desc"},
				},
			},
		},
	}
}

func TestContentDraftPublishProductFirst(t *testing.T) {
	router, _, publicCache := setupContentRouter(t)
	defer publicCache.Stop()

	// Missing doc → empty draft version 0
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/content/home/draft", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var draft map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &draft))
	assert.Equal(t, float64(0), draft["version"])

	// Create draft with If-Match: 0
	body, _ := json.Marshal(map[string]interface{}{"config": productHomeConfig()})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/admin/content/home/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "0")
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &draft))
	assert.Equal(t, float64(1), draft["version"])

	// Optimistic lock conflict
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/admin/content/home/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "0")
	router.ServeHTTP(w, req)
	assert.Equal(t, 409, w.Code)

	// Publish
	pubBody, _ := json.Marshal(map[string]interface{}{"expectedDraftVersion": 1})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/admin/content/home/publish", bytes.NewReader(pubBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())

	// Public read
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/public/content/home?locale=zh", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var pub map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pub))
	cfg := pub["config"].(map[string]interface{})
	hero := cfg["hero"].(map[string]interface{})
	assert.NotNil(t, hero["media"])

	// Warm cache then update+publish should invalidate
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/public/content/home?locale=zh", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, "HIT", w.Header().Get("X-Cache"))

	// Update draft v1→v2
	cfg2 := productHomeConfig()
	cfg2["hero"].(map[string]interface{})["title"] = map[string]interface{}{"zh": "新标题", "en": "New"}
	body, _ = json.Marshal(map[string]interface{}{"config": cfg2})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/admin/content/home/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "1")
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	pubBody, _ = json.Marshal(map[string]interface{}{"expectedDraftVersion": 2})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/admin/content/home/publish", bytes.NewReader(pubBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/public/content/home?locale=zh", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	// After invalidate, first read is MISS
	assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pub))
	cfg = pub["config"].(map[string]interface{})
	hero = cfg["hero"].(map[string]interface{})
	title := hero["title"].(map[string]interface{})
	assert.Equal(t, "新标题", title["zh"])
}

func TestContentUpdateDraftRejectsBilingualMediaCaption(t *testing.T) {
	router, _, publicCache := setupContentRouter(t)
	defer publicCache.Stop()

	cfg := productHomeConfig()
	cfg["hero"].(map[string]interface{})["media"] = map[string]interface{}{
		"url": "/x.png",
		"caption": map[string]interface{}{
			"zh": "中",
			"en": "En",
		},
	}
	body, _ := json.Marshal(map[string]interface{}{"config": cfg})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/content/home/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "0")
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "MEDIAREF_TYPE")
}

func TestContentValidateProductFirst(t *testing.T) {
	router, _, publicCache := setupContentRouter(t)
	defer publicCache.Stop()

	body, _ := json.Marshal(map[string]interface{}{"config": productHomeConfig()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/admin/content/home/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["valid"])
}

func TestContentPublishVersionMismatch(t *testing.T) {
	router, _, publicCache := setupContentRouter(t)
	defer publicCache.Stop()

	body, _ := json.Marshal(map[string]interface{}{"config": productHomeConfig()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/content/home/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "0")
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	pubBody, _ := json.Marshal(map[string]interface{}{"expectedDraftVersion": 99})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/admin/content/home/publish", bytes.NewReader(pubBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, 409, w.Code)
	assert.Contains(t, w.Body.String(), "CONFLICT_VERSION")
}

func TestContentInvalidPageKey(t *testing.T) {
	router, _, publicCache := setupContentRouter(t)
	defer publicCache.Stop()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/content/not-a-key/draft", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	_ = fmt.Sprintf("ok")
}
