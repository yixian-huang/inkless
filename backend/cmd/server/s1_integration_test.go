package main

// s1_integration_test.go — S1 spec §7 deferred backend integration tests.
//
// Exercises the publish→bootstrap round-trip end-to-end against a real
// in-memory SQLite DB.  These tests would have caught the "FindByKey
// returns &sc not nil" bug surfaced during smoke testing (fixed in 172cf5f).
//
// Tests:
//   TestS1_GlobalConfig_PublishRoundTrip       — happy-path PUT draft → publish → bootstrap
//   TestS1_GlobalConfig_RejectsInvalidSchema   — empty identity.name → 400
//   TestS1_Features_CreatesOnMissingRow        — PUT on empty table creates row (bug-catch)
//   TestS1_Features_VersionConflict409         — stale expectedDraftVersion → 409

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/db"
	authHandler "blotting-consultancy/internal/handler/auth"
	bootstrapHandler "blotting-consultancy/internal/handler/bootstrap"
	featuresHandler "blotting-consultancy/internal/handler/features"
	globalConfigHandler "blotting-consultancy/internal/handler/global_config"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	commentMod "blotting-consultancy/internal/modules/comment"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/seed"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/config"
)

// s1TestHarness holds the focused router and database for S1 tests.
type s1TestHarness struct {
	router      *gin.Engine
	database    *db.DB
	sharedCache *cache.Cache
}

// newS1Harness builds a minimal test router wiring only the routes exercised by the S1 tests:
//
//	/auth/login
//	/admin/global-config (GET, PUT /draft, POST /publish)
//	/admin/features      (GET, PUT /draft, POST /publish)
//	/public/bootstrap
//
// The SiteConfig table is migrated but NOT seeded — tests control the initial state.
func newS1Harness(t *testing.T) *s1TestHarness {
	t.Helper()

	database, err := db.Init(db.InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Migrate all tables needed by the exercised handlers.
	migrator := db.NewMigrator(database)
	err = migrator.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.ContentDocument{},
		&model.ContentVersion{},
		&model.InstalledTheme{},
		&model.PageView{},
		&model.Page{},
		&model.Article{},
		&model.Category{},
		&model.Tag{},
		&commentMod.Comment{},
		&model.Media{},
		&model.SiteConfig{},
	)
	require.NoError(t, err)

	// Seed admin user so tests can authenticate.
	ctx := context.Background()
	userRepo := repository.NewGormUserRepository(database.DB)
	contentDocRepo := repository.NewGormContentDocumentRepository(database.DB)
	installedThemeRepo := repository.NewGormInstalledThemeRepository(database.DB)
	pageRepo := repository.NewGormPageRepository(database.DB)
	themePageSvc := service.NewThemePageService(pageRepo)
	unifiedPageRepo := repository.NewGormUnifiedPageRepository(database.DB)
	pageTemplateRepo := repository.NewGormPageTemplateRepository(database.DB)
	seeder := seed.NewSeeder(userRepo, contentDocRepo, installedThemeRepo, themePageSvc, unifiedPageRepo, pageTemplateRepo, nil)
	require.NoError(t, seeder.SeedUsers(ctx))

	// Shared cache (short TTL — tests must issue bootstrap GET after publish).
	sharedCache := cache.New(time.Second)

	// Repository instances for handlers.
	siteConfigRepo := repository.NewGormSiteConfigRepository(database.DB)
	refreshTokenRepo := repository.NewGormRefreshTokenRepository(database.DB)

	// Handler instances.
	cfg := &config.Config{
		JWTSecret:        "s1-test-secret",
		JWTRefreshSecret: "s1-test-refresh-secret",
		Env:              "test",
	}
	authH := authHandler.NewHandler(userRepo, refreshTokenRepo, cfg)
	globalCfgH := globalConfigHandler.NewHandler(contentDocRepo, sharedCache)
	featuresH := featuresHandler.NewHandler(siteConfigRepo, sharedCache)
	bootstrapH := bootstrapHandler.NewHandler(contentDocRepo, installedThemeRepo, pageRepo, unifiedPageRepo, siteConfigRepo, sharedCache)

	// Router setup.
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(apierror.ErrorHandler())

	// Auth.
	authGroup := router.Group("/auth")
	authGroup.POST("/login", authH.Login)

	// Public.
	publicGroup := router.Group("/public")
	publicGroup.GET("/bootstrap", bootstrapH.PublicBootstrap)

	// Admin (JWT-protected).
	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.Auth(cfg.JWTSecret))
	adminGroup.Use(middleware.RequireAdminOrEditor())
	globalCfgH.RegisterRoutes(adminGroup)
	featuresH.RegisterRoutes(adminGroup)

	return &s1TestHarness{
		router:      router,
		database:    database,
		sharedCache: sharedCache,
	}
}

// adminLogin logs in as admin and returns the access token.
func (h *s1TestHarness) adminLogin(t *testing.T) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	h.router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, "login failed: %s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	token, ok := resp["accessToken"].(string)
	require.True(t, ok, "accessToken missing from login response")
	return token
}

// do is a tiny helper for authenticated JSON requests.
func (h *s1TestHarness) do(method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

// validGlobalConfigPayload returns a complete SiteConfigGlobal JSON payload suitable
// for PUT /admin/global-config/draft.
func validGlobalConfigPayload(expectedDraftVersion int) map[string]interface{} {
	return map[string]interface{}{
		"draftConfig": map[string]interface{}{
			"identity": map[string]interface{}{
				"name":          map[string]string{"zh": "测试站点", "en": "Test Site"},
				"tagline":       map[string]string{"zh": "留下足迹", "en": "Leave a Mark"},
				"localeMode":    "bilingual",
				"defaultLocale": "zh",
			},
			"brand": map[string]interface{}{
				"logo":         map[string]string{"light": "/logo.png"},
				"favicon":      "/favicon.ico",
				"ogImage":      "/og.png",
				"primaryColor": "#1a5f8f",
			},
			"author": map[string]interface{}{
				"name":    "测试团队",
				"socials": []interface{}{},
			},
			"footer": map[string]interface{}{},
			"seo":    map[string]interface{}{},
		},
		"expectedDraftVersion": expectedDraftVersion,
	}
}

// validFeaturesPayload returns a personal-blog features config.
func validFeaturesPayload(expectedDraftVersion int) map[string]interface{} {
	return map[string]interface{}{
		"draftConfig": map[string]interface{}{
			"publicPages": map[string]interface{}{
				"home":         true,
				"blog":         true,
				"contact":      false,
				"about":        false,
				"experts":      false,
				"coreServices": false,
				"advantages":   false,
				"cases":        false,
			},
			"blog": map[string]interface{}{
				"comments": true,
				"rss":      true,
			},
		},
		"expectedDraftVersion": expectedDraftVersion,
	}
}

// ─── Test 1 ────────────────────────────────────────────────────────────────

// TestS1_GlobalConfig_PublishRoundTrip verifies the full:
//
//	PUT /admin/global-config/draft → POST /admin/global-config/publish → GET /public/bootstrap
//
// round-trip, including verifying that the bilingual identity.name structure
// is preserved intact in the bootstrap response (not flattened to a single locale).
func TestS1_GlobalConfig_PublishRoundTrip(t *testing.T) {
	h := newS1Harness(t)
	defer h.database.Close()

	ctx := context.Background()

	// Seed a bare "global" content_document so FindByPageKey succeeds.
	contentDocRepo := repository.NewGormContentDocumentRepository(h.database.DB)
	require.NoError(t, contentDocRepo.Create(ctx, &model.ContentDocument{
		PageKey:          model.PageKeyGlobal,
		DraftConfig:      model.JSONMap{},
		DraftVersion:     1,
		PublishedConfig:  model.JSONMap{},
		PublishedVersion: 0,
	}))

	token := h.adminLogin(t)

	// Step 1: PUT draft with expectedDraftVersion=1.
	w := h.do("PUT", "/admin/global-config/draft", token, validGlobalConfigPayload(1))
	assert.Equal(t, 200, w.Code, "PUT draft: %s", w.Body.String())
	var putResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &putResp))
	assert.Equal(t, float64(2), putResp["draftVersion"], "expected draftVersion=2 after first PUT")

	// Step 2: POST publish.
	w = h.do("POST", "/admin/global-config/publish", token, nil)
	assert.Equal(t, 200, w.Code, "POST publish: %s", w.Body.String())
	var pubResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pubResp))
	assert.Equal(t, float64(1), pubResp["publishedVersion"], "expected publishedVersion=1")

	// Step 3: GET /public/bootstrap?locale=zh — cache was invalidated by publish.
	w = h.do("GET", "/public/bootstrap?locale=zh", "", nil)
	assert.Equal(t, 200, w.Code, "GET bootstrap: %s", w.Body.String())

	var boot map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &boot))

	globalCfgRaw, ok := boot["globalConfig"]
	require.True(t, ok, "bootstrap response must contain 'globalConfig'")
	globalCfg, ok := globalCfgRaw.(map[string]interface{})
	require.True(t, ok, "globalConfig must be an object")

	cfgRaw, ok := globalCfg["config"]
	require.True(t, ok, "globalConfig must contain 'config'")
	cfg, ok := cfgRaw.(map[string]interface{})
	require.True(t, ok, "globalConfig.config must be an object")

	identity, ok := cfg["identity"].(map[string]interface{})
	require.True(t, ok, "globalConfig.config.identity must be present")

	name, ok := identity["name"].(map[string]interface{})
	require.True(t, ok, "globalConfig.config.identity.name must be an object (bilingual)")

	// The bilingual structure must be preserved — both locales present.
	assert.Equal(t, "测试站点", name["zh"], "identity.name.zh should match what was written")
	assert.Equal(t, "Test Site", name["en"], "identity.name.en should be present (bilingual not flattened)")
}

// ─── Test 2 ────────────────────────────────────────────────────────────────

// TestS1_GlobalConfig_RejectsInvalidSchema verifies that PUT /admin/global-config/draft
// with an empty identity.name returns 400 with a message mentioning "identity.name".
func TestS1_GlobalConfig_RejectsInvalidSchema(t *testing.T) {
	h := newS1Harness(t)
	defer h.database.Close()

	ctx := context.Background()

	// Seed bare global doc.
	contentDocRepo := repository.NewGormContentDocumentRepository(h.database.DB)
	require.NoError(t, contentDocRepo.Create(ctx, &model.ContentDocument{
		PageKey:          model.PageKeyGlobal,
		DraftConfig:      model.JSONMap{},
		DraftVersion:     1,
		PublishedConfig:  model.JSONMap{},
		PublishedVersion: 0,
	}))

	token := h.adminLogin(t)

	// PUT with empty identity.name (both zh and en missing / empty).
	invalidPayload := map[string]interface{}{
		"draftConfig": map[string]interface{}{
			"identity": map[string]interface{}{
				"name":          map[string]string{"zh": "", "en": ""},
				"localeMode":    "bilingual",
				"defaultLocale": "zh",
			},
			"brand": map[string]interface{}{
				"logo":         map[string]string{"light": "/logo.png"},
				"favicon":      "/favicon.ico",
				"ogImage":      "/og.png",
				"primaryColor": "#000000",
			},
			"author": map[string]interface{}{
				"name":    "x",
				"socials": []interface{}{},
			},
			"footer": map[string]interface{}{},
			"seo":    map[string]interface{}{},
		},
		"expectedDraftVersion": 1,
	}

	w := h.do("PUT", "/admin/global-config/draft", token, invalidPayload)
	assert.Equal(t, 400, w.Code, "expected 400 for invalid schema; body: %s", w.Body.String())

	var errResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))

	// The error message should mention identity.name.
	errObj, _ := errResp["error"].(map[string]interface{})
	msg := fmt.Sprintf("%v", errObj["message"])
	assert.Contains(t, msg, "identity.name", "error message should mention identity.name; got: %q", msg)
}

// ─── Test 3 ────────────────────────────────────────────────────────────────

// TestS1_Features_CreatesOnMissingRow verifies that PUT /admin/features/draft
// on an EMPTY site_configs table creates the row (draftVersion=1), then publishes
// successfully, and the bootstrap endpoint reflects the published features.
//
// This is the smoke-test bug case: before the nil-check fix in the features handler,
// FindByKey returning (&sc, gorm.ErrRecordNotFound) would be misinterpreted and
// the PUT would fail instead of creating a new row.
func TestS1_Features_CreatesOnMissingRow(t *testing.T) {
	h := newS1Harness(t)
	defer h.database.Close()

	// Deliberately no seed for site_configs — table is empty.

	token := h.adminLogin(t)

	// Step 1: PUT features draft — must create the row, not fail.
	w := h.do("PUT", "/admin/features/draft", token, validFeaturesPayload(0))
	assert.Equal(t, 200, w.Code, "PUT features draft on empty table: %s", w.Body.String())
	var putResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &putResp))
	assert.Equal(t, float64(1), putResp["draftVersion"], "new row must start at draftVersion=1")

	// Step 2: POST publish.
	w = h.do("POST", "/admin/features/publish", token, nil)
	assert.Equal(t, 200, w.Code, "POST features publish: %s", w.Body.String())
	var pubResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pubResp))
	assert.Equal(t, float64(1), pubResp["publishedVersion"], "expected publishedVersion=1")

	// Step 3: GET /public/bootstrap?locale=zh — cache was flushed by publish.
	w = h.do("GET", "/public/bootstrap?locale=zh", "", nil)
	assert.Equal(t, 200, w.Code, "GET bootstrap: %s", w.Body.String())

	var boot map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &boot))

	featuresRaw, ok := boot["features"]
	require.True(t, ok, "bootstrap response must contain 'features'")
	features, ok := featuresRaw.(map[string]interface{})
	require.True(t, ok, "features must be an object")

	publicPages, ok := features["publicPages"].(map[string]interface{})
	require.True(t, ok, "features.publicPages must be present")

	assert.Equal(t, true, publicPages["home"], "features.publicPages.home should be true")
	assert.Equal(t, false, publicPages["about"], "features.publicPages.about should be false")
}

// ─── Test 4 ────────────────────────────────────────────────────────────────

// TestS1_Features_VersionConflict409 verifies that PUT /admin/features/draft
// with a stale expectedDraftVersion returns 409.
func TestS1_Features_VersionConflict409(t *testing.T) {
	h := newS1Harness(t)
	defer h.database.Close()

	ctx := context.Background()

	// Seed a features row at draftVersion=5.
	siteConfigRepo := repository.NewGormSiteConfigRepository(h.database.DB)
	require.NoError(t, siteConfigRepo.Upsert(ctx, &model.SiteConfig{
		Key:              model.SiteConfigKeyFeatures,
		DraftConfig:      model.JSONMap{"publicPages": map[string]interface{}{"home": true}},
		DraftVersion:     5,
		PublishedConfig:  model.JSONMap{},
		PublishedVersion: 0,
	}))

	token := h.adminLogin(t)

	// PUT with stale expectedDraftVersion=1 (current is 5).
	w := h.do("PUT", "/admin/features/draft", token, validFeaturesPayload(1))
	assert.Equal(t, 409, w.Code, "expected 409 for stale version; body: %s", w.Body.String())
}
