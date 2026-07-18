package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/db"
	authHandler "blotting-consultancy/internal/handler/auth"
	publicHandler "blotting-consultancy/internal/handler/public"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	commentMod "blotting-consultancy/internal/modules/comment"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/config"
	appLogger "blotting-consultancy/pkg/logger"
)

// setupTestRouter creates a test router with all routes wired
func setupTestRouter(t *testing.T) (*gin.Engine, *db.DB) {
	// Setup test database
	database, err := db.Init(db.InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Run migrations
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
	)
	require.NoError(t, err)

	// Initialize repositories
	userRepo := repository.NewGormUserRepository(database.DB)
	refreshTokenRepo := repository.NewGormRefreshTokenRepository(database.DB)
	contentDocRepo := repository.NewGormContentDocumentRepository(database.DB)
	pageViewRepo := repository.NewGormPageViewRepository(database.DB)

	// Initialize handlers
	cfg := &config.Config{
		JWTSecret:        "test-secret",
		JWTRefreshSecret: "test-refresh-secret",
		Env:              "test",
	}
	authHandlerInst := authHandler.NewHandler(userRepo, refreshTokenRepo, cfg)
	publicHandlerInst := publicHandler.NewHandler(contentDocRepo, pageViewRepo, nil, cache.New(60*time.Second))

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(apierror.ErrorHandler())

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := database.HealthCheck(ctx); err != nil {
			c.JSON(503, gin.H{"status": "unhealthy", "error": "database connection failed"})
			return
		}

		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Public routes
	publicGroup := router.Group("/public")
	{
		publicGroup.GET("/content/:pageKey", publicHandlerInst.GetPublicContent)
	}

	// Auth routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", authHandlerInst.Login)
		authGroup.POST("/refresh", authHandlerInst.Refresh)
		authGroup.POST("/logout", authHandlerInst.Logout)

		authProtected := authGroup.Group("")
		authProtected.Use(middleware.Auth(cfg.JWTSecret))
		{
			authProtected.GET("/me", authHandlerInst.Me)
		}
	}

	// Admin routes (no old content handler routes — unified page handler tested separately)
	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.Auth(cfg.JWTSecret))
	adminGroup.Use(middleware.RequireAdminOrEditor())
	{
		_ = adminGroup // routes registered above
	}

	return router, database
}

func TestHealthEndpoint(t *testing.T) {
	router, database := setupTestRouter(t)
	defer database.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestPublicRouteWiring(t *testing.T) {
	router, database := setupTestRouter(t)
	defer database.Close()

	// Create a test content document
	ctx := context.Background()
	docRepo := repository.NewGormContentDocumentRepository(database.DB)
	doc := &model.ContentDocument{
		PageKey:          model.PageKeyHome,
		PublishedConfig:  model.JSONMap{"title": "Test"},
		PublishedVersion: 1,
	}
	err := docRepo.Create(ctx, doc)
	require.NoError(t, err)

	// Test public content endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/public/content/home", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "home", response["pageKey"])
}

func TestAuthRouteWiring(t *testing.T) {
	router, database := setupTestRouter(t)
	defer database.Close()

	// Test login endpoint (should return 401 for invalid credentials)
	loginBody := map[string]string{
		"username": "nonexistent",
		"password": "wrong",
	}
	body, _ := json.Marshal(loginBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestAdminRouteAuthRequired(t *testing.T) {
	t.Skip("Admin routes are tested in integration tests with full handler wiring")
}

func TestMiddlewareOrdering(t *testing.T) {
	router, database := setupTestRouter(t)
	defer database.Close()

	// Test that recovery middleware works (router should not crash)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent-route", nil)
	router.ServeHTTP(w, req)

	// Should get 404, not panic
	assert.Equal(t, 404, w.Code)
}

func TestAdminSitesRoutesRemoved(t *testing.T) {
	router, database := setupTestRouter(t)
	defer database.Close()

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/admin/sites"},
		{method: http.MethodPost, path: "/admin/sites"},
		{method: http.MethodGet, path: "/admin/sites/1"},
		{method: http.MethodPut, path: "/admin/sites/1"},
		{method: http.MethodDelete, path: "/admin/sites/1"},
		{method: http.MethodGet, path: "/admin/sites/1/users"},
		{method: http.MethodPost, path: "/admin/sites/1/users"},
		{method: http.MethodDelete, path: "/admin/sites/1/users/2"},
		{method: http.MethodGet, path: "/admin/sites/1/export"},
		{method: http.MethodPost, path: "/admin/sites/import"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			router.ServeHTTP(response, request)
			assert.Equal(t, http.StatusNotFound, response.Code)
		})
	}
}

func TestRetiredAdminSitesHTMLPathIsRejectedBeforeSPAFallback(t *testing.T) {
	assert.True(t, isRetiredAdminSitesPath("/admin/sites"))
	assert.True(t, isRetiredAdminSitesPath("/admin/sites/1"))
	assert.False(t, isRetiredAdminSitesPath("/admin/site-config"))
	assert.False(t, isRetiredAdminSitesPath("/admin/sites-preview"))

	for _, path := range []string{"/admin/sites", "/admin/sites/1"} {
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest(http.MethodGet, path, nil)
		c.Request.Header.Set("Accept", "text/html,application/xhtml+xml")

		assert.True(t, rejectRetiredAdminSitesHTML(c))
		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusNotFound, c.Writer.Status())
	}
}

func TestFrontendFallbackReturns404ForRetiredAdminSites(t *testing.T) {
	frontendDir := t.TempDir()
	indexPath := frontendDir + "/index.html"
	require.NoError(t, os.WriteFile(indexPath, []byte("<html>spa</html>"), 0o600))

	router := gin.New()
	registerFrontendFallback(router, indexPath, nil, "https://example.com", nil)

	for _, path := range []string{"/admin/sites", "/admin/sites/1"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Accept", "text/html,application/xhtml+xml")
		router.ServeHTTP(response, request)

		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.NotContains(t, response.Body.String(), "<html>spa</html>")
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "<html>spa</html>")
}

func TestGracefulShutdownSetup(t *testing.T) {
	// This test just verifies the test setup works
	// Actual graceful shutdown is tested in integration tests
	// Set required env vars
	os.Setenv("DB_DSN", ":memory:")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	defer os.Unsetenv("DB_DSN")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("JWT_REFRESH_SECRET")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	log := appLogger.New(cfg.Env, nil)
	assert.NotNil(t, log)
}
