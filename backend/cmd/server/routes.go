package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	aiHandler "blotting-consultancy/internal/handler/ai"
	analyticsHandler "blotting-consultancy/internal/handler/analytics"
	articleHandler "blotting-consultancy/internal/handler/article"
	auditlogHandler "blotting-consultancy/internal/handler/auditlog"
	authHandler "blotting-consultancy/internal/handler/auth"
	bootstrapHandler "blotting-consultancy/internal/handler/bootstrap"
	categoryHandler "blotting-consultancy/internal/handler/category"
	chunkedUploadHandler "blotting-consultancy/internal/handler/chunked_upload"
	emailSettingsHandler "blotting-consultancy/internal/handler/email_settings"
	featuresHandler "blotting-consultancy/internal/handler/features"
	feedHandler "blotting-consultancy/internal/handler/feed"
	globalConfigHandler "blotting-consultancy/internal/handler/global_config"
	installedThemeHandler "blotting-consultancy/internal/handler/installed_theme"
	marketplaceHandler "blotting-consultancy/internal/handler/marketplace"
	mediaHandler "blotting-consultancy/internal/handler/media"
	mediaFolderHandler "blotting-consultancy/internal/handler/media_folder"
	menuHandler "blotting-consultancy/internal/handler/menu"
	migrationHandler "blotting-consultancy/internal/handler/migration"
	pageTemplateHandler "blotting-consultancy/internal/handler/page_template"
	pluginHandler "blotting-consultancy/internal/handler/plugin"
	publicHandler "blotting-consultancy/internal/handler/public"
	roleHandler "blotting-consultancy/internal/handler/role"
	searchhandler "blotting-consultancy/internal/handler/search"
	seoHandler "blotting-consultancy/internal/handler/seo"
	sitemapHandler "blotting-consultancy/internal/handler/sitemap"
	storageHandler "blotting-consultancy/internal/handler/storage"
	systemHandler "blotting-consultancy/internal/handler/system"
	tagHandler "blotting-consultancy/internal/handler/tag"
	themeHandler "blotting-consultancy/internal/handler/theme"
	themeExportHandler "blotting-consultancy/internal/handler/theme_export"
	translationHandler "blotting-consultancy/internal/handler/translation"
	unifiedPageHandler "blotting-consultancy/internal/handler/unified_page"
	userHandler "blotting-consultancy/internal/handler/user"
	wizardHandler "blotting-consultancy/internal/handler/wizard"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/module"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/seo"
	"blotting-consultancy/pkg/audit"
	"blotting-consultancy/pkg/config"
	"blotting-consultancy/pkg/metrics"

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
	Analytics      *analyticsHandler.Handler
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
}

// registerRoutes sets up all route groups, middleware, and endpoint registrations
// on the provided Gin engine.
func registerRoutes(router *gin.Engine, handlers *Handlers, deps *RouteDeps) {
	cfg := deps.Cfg

	// Version endpoint (public, no auth required)
	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":   Version,
			"buildTime": BuildTime,
			"gitCommit": GitCommit,
			"gitBranch": GitBranch,
		})
	})

	// Health endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// Check database connection
		if err := deps.Database.HealthCheck(ctx); err != nil {
			c.JSON(503, gin.H{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   Version,
			"buildTime": BuildTime,
			"gitCommit": GitCommit,
		})
	})

	// Metrics endpoint (no auth required, for operations dashboards)
	router.GET("/metrics", func(c *gin.Context) {
		m := metrics.Global()
		publishTotal, publishSuccess, publishFailure := m.GetPublishMetrics()
		validationTotal, validationFailures := m.GetValidationMetrics()
		rollbackTotal, rollbackSuccess, rollbackFailure, rollbackP95 := m.GetRollbackMetrics()
		publicGetTotal, publicGetSuccess, publicGetFailure, publicGetP95 := m.GetPublicGetMetrics()

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
		authProtected.Use(middleware.Auth(cfg.JWTSecret))
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
			if rejectRetiredAdminSitesHTML(c) {
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
	adminGroup.Use(middleware.Auth(cfg.JWTSecret))
	adminGroup.Use(middleware.AuditMutations(deps.AuditWriter))
	// Legacy middleware kept for backward compatibility with existing JWT tokens.
	// New RBAC permission checks are applied at the route-group level.
	adminGroup.Use(middleware.RequireAdminOrEditor())
	require := func(resource, action string) gin.HandlerFunc {
		return middleware.RequirePermission(resource, action, deps.UserRepo, deps.RBACCache)
	}

	// Register module routes
	deps.ModuleMgr.RegisterAllRoutes(publicGroup, adminGroup)

	{
		// Media management
		adminGroup.POST("/media/upload", require("media", "create"), handlers.Media.Upload)
		adminGroup.GET("/media", require("media", "read"), handlers.Media.List)
		adminGroup.DELETE("/media/:id", require("media", "delete"), handlers.Media.Delete)
		adminGroup.PUT("/media/:id/crop", require("media", "update"), handlers.Media.Recrop)
		adminGroup.PUT("/media/:id", require("media", "update"), handlers.Media.Rename)
		adminGroup.GET("/media/:id/usages", require("media", "read"), handlers.Media.GetUsages)

		// Analytics (requires analytics:read via RBAC)
		adminAnalytics := adminGroup.Group("")
		adminAnalytics.Use(require("analytics", "read"))
		{
			adminAnalytics.GET("/analytics/summary", handlers.Analytics.GetSummary)
		}

		// Article management
		adminGroup.GET("/articles", require("articles", "read"), handlers.Article.AdminList)
		adminGroup.GET("/articles/:id", require("articles", "read"), handlers.Article.AdminGetByID)
		adminGroup.POST("/articles", require("articles", "create"), handlers.Article.AdminCreate)
		adminGroup.PUT("/articles/:id", require("articles", "update"), handlers.Article.AdminUpdate)
		adminGroup.DELETE("/articles/:id", require("articles", "delete"), handlers.Article.AdminDelete)
		adminGroup.GET("/articles/:id/export", require("articles", "read"), handlers.Article.AdminExportMarkdown)
		adminGroup.POST("/articles/import", require("articles", "create"), handlers.Article.AdminImportMarkdown)

		scheduledPublications := adminGroup.Group("/scheduled-publications")
		scheduledPublications.Use(middleware.RequireAnyPermission(
			[]middleware.PermissionPair{
				{Resource: "articles", Action: "publish"},
				{Resource: "pages", Action: "publish"},
			},
			deps.UserRepo,
			deps.RBACCache,
		))
		{
			scheduledPublications.GET("", handlers.Scheduler.List)
			scheduledPublications.POST("", handlers.Scheduler.Schedule)
			scheduledPublications.PUT("/:id", handlers.Scheduler.Reschedule)
			scheduledPublications.DELETE("/:id", handlers.Scheduler.Cancel)
			scheduledPublications.POST("/:id/retry", handlers.Scheduler.Retry)
		}

		// Category management
		adminGroup.GET("/categories", require("categories", "read"), handlers.Category.List)
		adminGroup.GET("/categories/tree", require("categories", "read"), handlers.Category.ListTree)
		adminGroup.GET("/categories/:id", require("categories", "read"), handlers.Category.GetByID)
		adminGroup.POST("/categories", require("categories", "create"), handlers.Category.Create)
		adminGroup.PUT("/categories/:id", require("categories", "update"), handlers.Category.Update)
		adminGroup.DELETE("/categories/:id", require("categories", "delete"), handlers.Category.Delete)

		// Tag management
		adminGroup.GET("/tags", require("tags", "read"), handlers.Tag.List)
		adminGroup.POST("/tags", require("tags", "create"), handlers.Tag.Create)
		adminGroup.PUT("/tags/:id", require("tags", "update"), handlers.Tag.Update)
		adminGroup.DELETE("/tags/:id", require("tags", "delete"), handlers.Tag.Delete)

		// Menu management
		adminGroup.GET("/menus", require("menus", "read"), handlers.Menu.ListGroups)
		adminGroup.POST("/menus", require("menus", "create"), handlers.Menu.CreateGroup)
		adminGroup.GET("/menus/:id", require("menus", "read"), handlers.Menu.GetGroup)
		adminGroup.PUT("/menus/:id", require("menus", "update"), handlers.Menu.UpdateGroup)
		adminGroup.DELETE("/menus/:id", require("menus", "delete"), handlers.Menu.DeleteGroup)
		adminGroup.PUT("/menus/:id/primary", require("menus", "update"), handlers.Menu.SetPrimary)
		adminGroup.POST("/menus/:id/items", require("menus", "update"), handlers.Menu.CreateItem)
		adminGroup.PUT("/menus/:id/items/:itemId", require("menus", "update"), handlers.Menu.UpdateItem)
		adminGroup.DELETE("/menus/:id/items/:itemId", require("menus", "update"), handlers.Menu.DeleteItem)
		adminGroup.PUT("/menus/:id/items/reorder", require("menus", "update"), handlers.Menu.ReorderItems)

		// Audit logs (requires audit_logs:read via RBAC)
		adminAudit := adminGroup.Group("")
		adminAudit.Use(require("audit_logs", "read"))
		{
			adminAudit.GET("/audit-logs", handlers.AuditLog.List)
		}

		// Theme token management (existing)
		adminGroup.GET("/theme", require("themes", "read"), handlers.Theme.AdminGet)
		adminGroup.PUT("/theme", require("themes", "update"), handlers.Theme.AdminUpdate)

		// Installed theme management
		adminGroup.GET("/themes", require("themes", "read"), handlers.InstalledTheme.AdminList)
		adminGroup.GET("/themes/:id", require("themes", "read"), handlers.InstalledTheme.AdminGetByID)
		adminGroup.POST("/themes", require("themes", "create"), handlers.InstalledTheme.AdminCreate)
		adminGroup.PUT("/themes/:id", require("themes", "update"), handlers.InstalledTheme.AdminUpdate)
		adminGroup.DELETE("/themes/:id", require("themes", "delete"), handlers.InstalledTheme.AdminDelete)
		adminGroup.PUT("/themes/:id/activate", require("themes", "manage"), handlers.InstalledTheme.AdminActivate)

		// Email settings management
		adminGroup.GET("/email-settings", require("settings", "read"), handlers.EmailSettings.HandleGet)
		adminGroup.PUT("/email-settings", require("settings", "manage"), handlers.EmailSettings.HandleUpdate)
		adminGroup.POST("/email-settings/test", require("settings", "manage"), handlers.EmailSettings.HandleTest)

		// Global config (branding / identity / SEO defaults)
		globalConfigAdmin := adminGroup.Group("")
		globalConfigAdmin.Use(require("settings", "manage"))
		handlers.GlobalConfig.RegisterRoutes(globalConfigAdmin)

		// Features (route gates, blog comments/rss toggles)
		featuresAdmin := adminGroup.Group("")
		featuresAdmin.Use(require("settings", "manage"))
		handlers.Features.RegisterRoutes(featuresAdmin)

		// User management
		adminUsers := adminGroup.Group("/users")
		{
			adminUsers.GET("", require("users", "read"), handlers.User.List)
			adminUsers.GET("/:id", require("users", "read"), handlers.User.GetByID)
			adminUsers.POST("", require("users", "create"), handlers.User.Create)
			adminUsers.PUT("/:id", require("users", "update"), handlers.User.Update)
			adminUsers.DELETE("/:id", require("users", "delete"), handlers.User.Delete)
		}

		// RBAC Role management
		adminRoles := adminGroup.Group("/roles")
		{
			adminRoles.GET("", require("roles", "read"), handlers.Role.List)
			adminRoles.GET("/:id", require("roles", "read"), handlers.Role.GetByID)
			adminRoles.POST("", require("roles", "create"), handlers.Role.Create)
			adminRoles.PUT("/:id", require("roles", "update"), handlers.Role.Update)
			adminRoles.DELETE("/:id", require("roles", "delete"), handlers.Role.Delete)
			adminRoles.POST("/assign", require("roles", "manage"), handlers.Role.AssignRole)
			adminRoles.POST("/unassign", require("roles", "manage"), handlers.Role.UnassignRole)
			adminRoles.GET("/user/:userId", require("roles", "read"), handlers.Role.GetUserRoles)
		}

		// Permission listing (requires roles:read via RBAC)
		adminGroup.GET("/permissions", require("roles", "read"), handlers.Role.ListPermissions)

		// AI Site Building Wizard
		adminGroup.POST("/wizard/generate-plan", require("pages", "create"), handlers.Wizard.GeneratePlan)
		adminGroup.POST("/wizard/apply-plan", require("pages", "create"), handlers.Wizard.ApplyPlan)
		adminGroup.POST("/wizard/suggest-colors", require("pages", "create"), handlers.Wizard.SuggestColors)
		adminGroup.POST("/wizard/generate-content", require("pages", "create"), handlers.Wizard.GenerateContent)

		// Marketplace (plugin/theme registry)
		adminGroup.GET("/marketplace/items", require("plugins", "read"), handlers.Marketplace.AdminListItems)
		adminGroup.GET("/marketplace/installed", require("plugins", "read"), handlers.Marketplace.AdminListInstalled)
		adminGroup.POST("/marketplace/items", require("plugins", "manage"), handlers.Marketplace.AdminRegisterItem)
		adminGroup.GET("/marketplace/items/:slug", require("plugins", "read"), handlers.Marketplace.AdminGetItem)
		adminGroup.POST("/marketplace/items/:slug/install", require("plugins", "manage"), handlers.Marketplace.AdminInstallItem)
		adminGroup.PUT("/marketplace/items/:slug/update", require("plugins", "manage"), handlers.Marketplace.AdminUpdateItem)
		adminGroup.DELETE("/marketplace/items/:slug", require("plugins", "manage"), handlers.Marketplace.AdminUninstallItem)
		adminGroup.POST("/marketplace/items/:slug/versions", require("plugins", "manage"), handlers.Marketplace.AdminAddVersion)

		// External plugin lifecycle
		adminGroup.GET("/plugins", require("plugins", "read"), handlers.Plugin.List)
		adminGroup.POST("/plugins/install", require("system", "manage"), handlers.Plugin.Install)
		adminGroup.POST("/plugins/:id/enable", require("system", "manage"), handlers.Plugin.Enable)
		adminGroup.POST("/plugins/:id/disable", require("system", "manage"), handlers.Plugin.Disable)
		adminGroup.DELETE("/plugins/:id", require("system", "manage"), handlers.Plugin.Uninstall)
		adminGroup.PUT("/plugins/:id/settings", require("system", "manage"), handlers.Plugin.UpdateSettings)
		adminGroup.POST("/plugins/test-notification", require("system", "manage"), handlers.Plugin.TestNotification)

		// AI provider management
		adminGroup.POST("/ai/chat", require("settings", "manage"), handlers.AI.Chat)
		adminGroup.POST("/ai/summarize", require("settings", "manage"), handlers.AI.Summarize)
		adminGroup.POST("/ai/suggest-titles", require("settings", "manage"), handlers.AI.SuggestTitles)
		adminGroup.POST("/ai/suggest-tags", require("settings", "manage"), handlers.AI.SuggestTags)
		adminGroup.POST("/ai/complete", require("settings", "manage"), handlers.AI.Complete)
		adminGroup.GET("/ai/config", require("settings", "manage"), handlers.AI.GetConfig)
		adminGroup.PUT("/ai/config", require("settings", "manage"), handlers.AI.UpdateConfig)
		adminGroup.POST("/ai/config/test", require("settings", "manage"), handlers.AI.TestConfig)

		// Chunked upload
		adminGroup.POST("/media/upload/init", require("media", "create"), handlers.ChunkedUpload.InitUpload)
		adminGroup.POST("/media/upload/:uploadId/chunk", require("media", "create"), handlers.ChunkedUpload.UploadChunk)
		adminGroup.POST("/media/upload/:uploadId/complete", require("media", "create"), handlers.ChunkedUpload.CompleteUpload)

		// Media folders
		adminGroup.GET("/media/folders", require("media", "read"), handlers.MediaFolder.ListTree)
		adminGroup.POST("/media/folders", require("media", "create"), handlers.MediaFolder.Create)
		adminGroup.PUT("/media/folders/:id", require("media", "update"), handlers.MediaFolder.Rename)
		adminGroup.DELETE("/media/folders/:id", require("media", "delete"), handlers.MediaFolder.Delete)
		adminGroup.PUT("/media/:id/move", require("media", "update"), handlers.MediaFolder.MoveMedia)

		// Data migration (import from WordPress, Halo, Markdown)
		adminGroup.POST("/migration/import", require("system", "manage"), handlers.Migration.Import)
		adminGroup.GET("/migration/jobs", require("system", "manage"), handlers.Migration.ListJobs)
		adminGroup.GET("/migration/jobs/:jobId", require("system", "manage"), handlers.Migration.GetJob)
		adminGroup.POST("/migration/jobs/:jobId/retry", require("system", "manage"), handlers.Migration.RetryJob)
		adminGroup.GET("/migration/jobs/:jobId/stream", require("system", "manage"), handlers.Migration.StreamProgress)

		// Storage configuration
		adminGroup.GET("/storage/config", require("settings", "manage"), handlers.Storage.GetConfig)
		adminGroup.PUT("/storage/config", require("settings", "manage"), handlers.Storage.UpdateConfig)
		adminGroup.POST("/storage/test", require("settings", "manage"), handlers.Storage.TestConnection)

		// System status
		adminGroup.GET("/system/status", require("system", "manage"), handlers.System.GetStatus)

		// Translation & glossary
		adminGroup.POST("/translate", require("settings", "manage"), handlers.Translation.Translate)
		adminGroup.POST("/translate/batch", require("settings", "manage"), handlers.Translation.BatchTranslate)
		adminGroup.POST("/translate/article/:id", require("settings", "manage"), handlers.Translation.TranslateArticle)
		adminGroup.GET("/glossary", require("settings", "manage"), handlers.Translation.GlossaryList)
		adminGroup.POST("/glossary", require("settings", "manage"), handlers.Translation.GlossaryCreate)
		adminGroup.PUT("/glossary/:id", require("settings", "manage"), handlers.Translation.GlossaryUpdate)
		adminGroup.DELETE("/glossary/:id", require("settings", "manage"), handlers.Translation.GlossaryDelete)

		// Page management (unified page system)
		adminGroup.GET("/pages", require("pages", "read"), handlers.UnifiedPage.AdminList)
		adminGroup.GET("/pages/:id", require("pages", "read"), handlers.UnifiedPage.AdminGetByID)
		adminGroup.POST("/pages", require("pages", "create"), handlers.UnifiedPage.AdminCreate)
		adminGroup.PUT("/pages/:id", require("pages", "update"), handlers.UnifiedPage.AdminUpdate)
		adminGroup.GET("/pages/:id/draft", require("pages", "read"), handlers.UnifiedPage.AdminGetDraft)
		adminGroup.PUT("/pages/:id/draft", require("pages", "update"), handlers.UnifiedPage.AdminUpdateDraft)
		adminGroup.POST("/pages/:id/publish", require("pages", "publish"), handlers.UnifiedPage.AdminPublish)
		adminGroup.POST("/pages/:id/unpublish", require("pages", "publish"), handlers.UnifiedPage.AdminUnpublish)
		adminGroup.POST("/pages/:id/rollback", require("pages", "publish"), handlers.UnifiedPage.AdminRollback)
		adminGroup.GET("/pages/:id/versions", require("pages", "read"), handlers.UnifiedPage.AdminListVersions)
		adminGroup.GET("/pages/:id/versions/:version", require("pages", "read"), handlers.UnifiedPage.AdminGetVersionDetail)
		adminGroup.DELETE("/pages/:id", require("pages", "delete"), handlers.UnifiedPage.AdminDelete)

		// Page template management
		adminGroup.GET("/templates", require("pages", "read"), handlers.PageTemplate.List)
		adminGroup.POST("/templates", require("pages", "create"), handlers.PageTemplate.Create)
		adminGroup.PUT("/templates/:id", require("pages", "update"), handlers.PageTemplate.Update)
		adminGroup.DELETE("/templates/:id", require("pages", "delete"), handlers.PageTemplate.Delete)
		adminGroup.POST("/templates/:id/duplicate", require("pages", "create"), handlers.PageTemplate.Duplicate)

		// Theme export/import
		adminGroup.POST("/theme-packages/export", require("themes", "manage"), handlers.ThemeExport.Export)
		adminGroup.POST("/theme-packages/import", require("themes", "manage"), handlers.ThemeExport.Import)
		adminGroup.GET("/theme-packages", require("themes", "manage"), handlers.ThemeExport.List)
		adminGroup.PUT("/theme-packages/:id/apply", require("themes", "manage"), handlers.ThemeExport.Apply)
	}

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
		router.StaticFile("/favicon.ico", filepath.Join(cfg.FrontendDir, "favicon.ico"))

		// SPA fallback: non-API GET requests return index.html with SEO meta
		indexHTML := filepath.Join(cfg.FrontendDir, "index.html")
		registerFrontendFallback(router, indexHTML, seoRenderer, cfg.BaseURL, deps.ContentDocRepo)
	}
}

func registerFrontendFallback(
	router *gin.Engine,
	indexHTML string,
	renderer *seo.Renderer,
	baseURL string,
	contentDocRepo repository.ContentDocumentRepository,
) {
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if isRetiredAdminSitesPath(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if c.Request.Method == http.MethodGet &&
			!strings.HasPrefix(path, "/public/") &&
			!strings.HasPrefix(path, "/auth/") &&
			!strings.HasPrefix(path, "/uploads/") &&
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

func isRetiredAdminSitesPath(path string) bool {
	return path == "/admin/sites" || strings.HasPrefix(path, "/admin/sites/")
}

func rejectRetiredAdminSitesHTML(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet ||
		!strings.Contains(c.GetHeader("Accept"), "text/html") ||
		!isRetiredAdminSitesPath(c.Request.URL.Path) {
		return false
	}
	c.Status(http.StatusNotFound)
	c.Abort()
	return true
}
