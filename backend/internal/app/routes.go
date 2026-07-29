package app

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	aiHandler "github.com/yixian-huang/inkless/backend/internal/handler/ai"
	analyticsHandler "github.com/yixian-huang/inkless/backend/internal/handler/analytics"
	articleHandler "github.com/yixian-huang/inkless/backend/internal/handler/article"
	auditlogHandler "github.com/yixian-huang/inkless/backend/internal/handler/auditlog"
	authHandler "github.com/yixian-huang/inkless/backend/internal/handler/auth"
	bootstrapHandler "github.com/yixian-huang/inkless/backend/internal/handler/bootstrap"
	categoryHandler "github.com/yixian-huang/inkless/backend/internal/handler/category"
	chunkedUploadHandler "github.com/yixian-huang/inkless/backend/internal/handler/chunked_upload"
	dashboardHandler "github.com/yixian-huang/inkless/backend/internal/handler/dashboard"
	emailSettingsHandler "github.com/yixian-huang/inkless/backend/internal/handler/email_settings"
	extensionsHandler "github.com/yixian-huang/inkless/backend/internal/handler/extensions"
	featuresHandler "github.com/yixian-huang/inkless/backend/internal/handler/features"
	feedHandler "github.com/yixian-huang/inkless/backend/internal/handler/feed"
	globalConfigHandler "github.com/yixian-huang/inkless/backend/internal/handler/global_config"
	installedThemeHandler "github.com/yixian-huang/inkless/backend/internal/handler/installed_theme"
	marketplaceHandler "github.com/yixian-huang/inkless/backend/internal/handler/marketplace"
	mediaHandler "github.com/yixian-huang/inkless/backend/internal/handler/media"
	mediaFolderHandler "github.com/yixian-huang/inkless/backend/internal/handler/media_folder"
	menuHandler "github.com/yixian-huang/inkless/backend/internal/handler/menu"
	migrationHandler "github.com/yixian-huang/inkless/backend/internal/handler/migration"
	pageTemplateHandler "github.com/yixian-huang/inkless/backend/internal/handler/page_template"
	pluginHandler "github.com/yixian-huang/inkless/backend/internal/handler/plugin"
	publicHandler "github.com/yixian-huang/inkless/backend/internal/handler/public"
	roleHandler "github.com/yixian-huang/inkless/backend/internal/handler/role"
	searchhandler "github.com/yixian-huang/inkless/backend/internal/handler/search"
	seoHandler "github.com/yixian-huang/inkless/backend/internal/handler/seo"
	sitemapHandler "github.com/yixian-huang/inkless/backend/internal/handler/sitemap"
	storageHandler "github.com/yixian-huang/inkless/backend/internal/handler/storage"
	systemHandler "github.com/yixian-huang/inkless/backend/internal/handler/system"
	tagHandler "github.com/yixian-huang/inkless/backend/internal/handler/tag"
	themeHandler "github.com/yixian-huang/inkless/backend/internal/handler/theme"
	themeExportHandler "github.com/yixian-huang/inkless/backend/internal/handler/theme_export"
	translationHandler "github.com/yixian-huang/inkless/backend/internal/handler/translation"
	unifiedPageHandler "github.com/yixian-huang/inkless/backend/internal/handler/unified_page"
	userHandler "github.com/yixian-huang/inkless/backend/internal/handler/user"
	wizardHandler "github.com/yixian-huang/inkless/backend/internal/handler/wizard"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/db"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/module"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/seo"
	"github.com/yixian-huang/inkless/backend/pkg/audit"
	"github.com/yixian-huang/inkless/backend/pkg/brand"
	"github.com/yixian-huang/inkless/backend/pkg/config"
	"github.com/yixian-huang/inkless/backend/pkg/metrics"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handlers holds all initialized HTTP handlers.
type Handlers struct {
	Auth           *authHandler.Handler
	Article        *articleHandler.Handler
	Public         *publicHandler.Handler
	Bootstrap      *bootstrapHandler.Handler
	Media          *mediaHandler.Handler
	APIKey         interface {
		List(*gin.Context)
		Create(*gin.Context)
		Revoke(*gin.Context)
	}
	// Agent multi-site discovery (whoami); optional for tests that omit it.
	Agent interface {
		Whoami(*gin.Context)
	}
	Analytics      *analyticsHandler.Handler
	Dashboard      *dashboardHandler.Handler
	Category       *categoryHandler.Handler
	Tag            *tagHandler.Handler
	Menu           *menuHandler.Handler
	AuditLog       *auditlogHandler.Handler
	Sitemap        *sitemapHandler.Handler
	Feed           *feedHandler.Handler
	Theme          *themeHandler.Handler
	InstalledTheme *installedThemeHandler.Handler
	EmailSettings  *emailSettingsHandler.Handler
	Features       *featuresHandler.Handler
	GlobalConfig   *globalConfigHandler.Handler
	User           *userHandler.Handler
	SEO            *seoHandler.Handler
	Search         *searchhandler.Handler
	Role           *roleHandler.Handler
	Marketplace    *marketplaceHandler.Handler
	Extensions     *extensionsHandler.Handler
	Plugin         *pluginHandler.Handler
	Wizard         *wizardHandler.Handler
	AI             *aiHandler.Handler
	ChunkedUpload  *chunkedUploadHandler.Handler
	MediaFolder    *mediaFolderHandler.Handler
	Migration      *migrationHandler.Handler
	Storage        *storageHandler.Handler
	System         *systemHandler.Handler
	Translation    *translationHandler.Handler
	UnifiedPage    *unifiedPageHandler.Handler
	Scheduler      interface {
		List(*gin.Context)
		Schedule(*gin.Context)
		Reschedule(*gin.Context)
		Cancel(*gin.Context)
		Retry(*gin.Context)
	}
	PageTemplate *pageTemplateHandler.Handler
	ThemeExport  *themeExportHandler.Handler
}

// RouteDeps holds dependencies needed by route middleware.
type RouteDeps struct {
	UserRepo       repository.UserRepository
	RBACCache      *cache.Cache
	Cfg            *config.Config
	Database       *db.DB
	ModuleMgr      *module.Manager
	ContentDocRepo repository.ContentDocumentRepository
	AuditWriter    audit.Writer
	Build          BuildInfo
	// APIKeyAuth optional long-lived token authenticator (PicGo / local agents).
	APIKeyAuth middleware.APIKeyAuthenticator
}

// registerRoutes sets up all route groups, middleware, and endpoint registrations
// on the provided Gin engine.
func registerRoutes(router *gin.Engine, handlers *Handlers, deps *RouteDeps) {
	cfg := deps.Cfg
	build := deps.Build

	// Version endpoint (public, no auth required)
	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":   build.Version,
			"buildTime": build.BuildTime,
			"gitCommit": build.GitCommit,
			"gitBranch": build.GitBranch,
		})
	})

	// Health endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// Check database connection
		if err := deps.Database.HealthCheck(ctx); err != nil {
			c.JSON(503, gin.H{
				"service": brand.APIService,
				"status":  "unhealthy",
				"error":   "database connection failed",
			})
			return
		}

		c.JSON(200, gin.H{
			"service":   brand.APIService,
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   build.Version,
			"buildTime": build.BuildTime,
			"gitCommit": build.GitCommit,
		})
	})

	// Metrics endpoint (no auth required, for operations dashboards)
	router.GET("/metrics", func(c *gin.Context) {
		m := metrics.Global()
		publishTotal, publishSuccess, publishFailure := m.GetPublishMetrics()
		validationTotal, validationFailures := m.GetValidationMetrics()
		rollbackTotal, rollbackSuccess, rollbackFailure, rollbackP95 := m.GetRollbackMetrics()
		publicGetTotal, publicGetSuccess, publicGetFailure, publicGetP95 := m.GetPublicGetMetrics()
		httpTotal, http2xx, http4xx, http5xx, httpSlow, httpP95 := m.GetHTTPMetrics()

		c.JSON(200, gin.H{
			"publish": gin.H{
				"total":   publishTotal,
				"success": publishSuccess,
				"failure": publishFailure,
			},
			"validation": gin.H{
				"total":    validationTotal,
				"failures": validationFailures,
			},
			"rollback": gin.H{
				"total":       rollbackTotal,
				"success":     rollbackSuccess,
				"failure":     rollbackFailure,
				"latency_p95": rollbackP95.Milliseconds(),
			},
			"http": gin.H{
				"total":       httpTotal,
				"2xx":         http2xx,
				"4xx":         http4xx,
				"5xx":         http5xx,
				"slow":        httpSlow,
				"latency_p95": httpP95.Milliseconds(),
			},
			"public_get": gin.H{
				"total":       publicGetTotal,
				"success":     publicGetSuccess,
				"failure":     publicGetFailure,
				"latency_p95": publicGetP95.Milliseconds(),
			},
		})
	})

	// Swagger API documentation (no auth required)
	router.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Sitemap (no auth required)
	router.GET("/sitemap.xml", handlers.Sitemap.GetSitemap)

	// RSS feed (no auth required)
	if handlers.Feed != nil {
		router.GET("/feed.xml", handlers.Feed.GetFeed)
	}

	// Robots.txt (no auth required)
	router.GET("/robots.txt", handlers.SEO.GetRobotsTxt)

	// Public routes (no auth required)
	publicGroup := router.Group("/public")
	publicGroup.Use(middleware.PublicRateLimit())
	{
		publicGroup.GET("/bootstrap", handlers.Bootstrap.PublicBootstrap)
		publicGroup.GET("/content/:pageKey", handlers.Public.GetPublicContent)

		// Public article routes
		publicGroup.GET("/articles", handlers.Article.PublicList)
		publicGroup.GET("/articles/:slug", handlers.Article.PublicGetBySlug)

		// Public category routes
		publicGroup.GET("/categories", handlers.Category.PublicList)
		publicGroup.GET("/categories/:slug", handlers.Category.PublicGetBySlug)

		// Public tag routes
		publicGroup.GET("/tags", handlers.Tag.PublicList)
		publicGroup.GET("/tags/:slug", handlers.Tag.PublicGetBySlug)

		// Public menu route
		publicGroup.GET("/menu", handlers.Menu.PublicGetPrimary)

		// Public theme route
		publicGroup.GET("/theme", handlers.Theme.PublicGet)

		// Public active theme route
		publicGroup.GET("/active-theme", handlers.InstalledTheme.PublicGetActive)

		// Unified pages (replaces old page routes)
		publicGroup.GET("/pages", handlers.UnifiedPage.PublicList)
		publicGroup.GET("/pages/:slug", handlers.UnifiedPage.PublicGetBySlug)
	}

	// Auth routes (no auth middleware, but handlers validate credentials)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", middleware.AuditLogin(deps.AuditWriter), middleware.LoginRateLimit(), handlers.Auth.Login)
		authGroup.POST("/refresh", handlers.Auth.Refresh)
		authGroup.POST("/logout", handlers.Auth.Logout)

		// Protected auth routes
		authProtected := authGroup.Group("")
		authProtected.Use(middleware.Auth(cfg.JWTSecret, deps.APIKeyAuth))
		{
			authProtected.GET("/me", handlers.Auth.Me)
		}
	}

	// Initialize SEO renderer when FRONTEND_DIR is configured
	var seoRenderer *seo.Renderer
	if cfg.FrontendDir != "" {
		seoIndexPath := filepath.Join(cfg.FrontendDir, "index.html")
		var err error
		seoRenderer, err = seo.NewRenderer(seoIndexPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to create SEO renderer: %v", err))
		}
	}

	// Admin routes (require authentication and authorization)
	adminGroup := router.Group("/admin")
	// SPA fallback: if FRONTEND_DIR is set and the browser asks for HTML,
	// serve index.html instead of requiring auth (the SPA handles its own auth).
	if cfg.FrontendDir != "" {
		indexPath := filepath.Join(cfg.FrontendDir, "index.html")
		adminGroup.Use(func(c *gin.Context) {
			if RejectRetiredAdminSitesHTML(c) {
				return
			}
			accept := c.GetHeader("Accept")
			if c.Request.Method == "GET" && strings.Contains(accept, "text/html") {
				if !serveSPAWithMeta(c, seoRenderer, cfg.BaseURL, deps.ContentDocRepo) {
					c.File(indexPath)
					c.Abort()
				}
				return
			}
			c.Next()
		})
	}
	adminGroup.Use(middleware.Auth(cfg.JWTSecret, deps.APIKeyAuth))
	adminGroup.Use(middleware.AuditMutations(deps.AuditWriter))
	// Load RBAC user (roles + permissions) once per request. JWT already
	// validates admin|editor; RequireAdminOrEditor is redundant and would
	// block future custom JWT roles. Per-route require() only checks perms.
	adminGroup.Use(middleware.LoadRBACUser(deps.UserRepo, deps.RBACCache))
	require := func(resource, action string) gin.HandlerFunc {
		return middleware.RequirePermission(resource, action, deps.UserRepo, deps.RBACCache)
	}

	// Register module routes
	deps.ModuleMgr.RegisterAllRoutes(publicGroup, adminGroup)

	// Domain-split admin route tables (see routes_admin.go).
	registerAdminRoutes(adminGroup, handlers, require, deps.UserRepo, deps.RBACCache)

	// SEO routes (public + admin)
	seoAdmin := adminGroup.Group("")
	seoAdmin.Use(require("settings", "manage"))
	handlers.SEO.RegisterRoutes(publicGroup, seoAdmin)

	// Search routes (public + admin)
	searchAdmin := adminGroup.Group("")
	searchAdmin.Use(require("settings", "manage"))
	handlers.Search.RegisterRoutes(publicGroup, searchAdmin)

	// Serve uploaded files statically
	router.Static("/uploads", cfg.UploadDir)

	// Serve frontend static assets when FRONTEND_DIR is configured
	if cfg.FrontendDir != "" {
		router.Static("/assets", filepath.Join(cfg.FrontendDir, "assets"))
		router.Static("/images", filepath.Join(cfg.FrontendDir, "images"))
		// Brand kit (favicon, logos, OG) from Vite public/brand — not SPA-routed
		router.Static("/brand", filepath.Join(cfg.FrontendDir, "brand"))
		// Official theme marketplace index + UMD mirrors (Vite public/marketplace)
		router.Static("/marketplace", filepath.Join(cfg.FrontendDir, "marketplace"))
		router.StaticFile("/favicon.ico", filepath.Join(cfg.FrontendDir, "favicon.ico"))

		// SPA fallback: non-API GET requests return index.html with SEO meta
		indexHTML := filepath.Join(cfg.FrontendDir, "index.html")
		RegisterFrontendFallback(router, indexHTML, seoRenderer, cfg.BaseURL, deps.ContentDocRepo)
	}
}

// RegisterFrontendFallback installs SPA NoRoute handling for non-API GETs.
func RegisterFrontendFallback(
	router *gin.Engine,
	indexHTML string,
	renderer *seo.Renderer,
	baseURL string,
	contentDocRepo repository.ContentDocumentRepository,
) {
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if IsRetiredAdminSitesPath(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if c.Request.Method == http.MethodGet &&
			!strings.HasPrefix(path, "/public/") &&
			!strings.HasPrefix(path, "/auth/") &&
			!strings.HasPrefix(path, "/uploads/") &&
			!strings.HasPrefix(path, "/brand/") &&
			path != "/health" &&
			path != "/version" &&
			path != "/metrics" &&
			path != "/sitemap.xml" &&
			path != "/robots.txt" {
			if !serveSPAWithMeta(c, renderer, baseURL, contentDocRepo) {
				http.ServeFile(c.Writer, c.Request, indexHTML)
				c.Abort()
			}
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})
}

// IsRetiredAdminSitesPath reports multi-site admin paths removed from this product.
func IsRetiredAdminSitesPath(path string) bool {
	return path == "/admin/sites" || strings.HasPrefix(path, "/admin/sites/")
}

// RejectRetiredAdminSitesHTML aborts HTML GETs to retired /admin/sites paths.
func RejectRetiredAdminSitesHTML(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet ||
		!strings.Contains(c.GetHeader("Accept"), "text/html") ||
		!IsRetiredAdminSitesPath(c.Request.URL.Path) {
		return false
	}
	c.Status(http.StatusNotFound)
	c.Abort()
	return true
}
