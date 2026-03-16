package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/backup"
	"blotting-consultancy/internal/seo"
	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/db/migrations"
	_ "blotting-consultancy/docs/swagger" // swagger docs
	"blotting-consultancy/internal/eventbus"
	analyticsHandler "blotting-consultancy/internal/handler/analytics"
	articleHandler "blotting-consultancy/internal/handler/article"
	roleHandler "blotting-consultancy/internal/handler/role"
	wizardHandler "blotting-consultancy/internal/handler/wizard"
	auditlogHandler "blotting-consultancy/internal/handler/auditlog"
	authHandler "blotting-consultancy/internal/handler/auth"
	backupHandler "blotting-consultancy/internal/handler/backup"
	categoryHandler "blotting-consultancy/internal/handler/category"
	commentHandler "blotting-consultancy/internal/handler/comment"
	// contentHandler removed — replaced by unified page handler
	mediaHandler "blotting-consultancy/internal/handler/media"
	// pageHandler removed — replaced by unified page handler
	publicHandler "blotting-consultancy/internal/handler/public"
	sitemapHandler "blotting-consultancy/internal/handler/sitemap"
	tagHandler "blotting-consultancy/internal/handler/tag"
	bootstrapHandler "blotting-consultancy/internal/handler/bootstrap"
	formSubmissionHandler "blotting-consultancy/internal/handler/form_submission"
	installedThemeHandler "blotting-consultancy/internal/handler/installed_theme"
	marketplaceHandler "blotting-consultancy/internal/handler/marketplace"
	menuHandler "blotting-consultancy/internal/handler/menu"
	themeHandler "blotting-consultancy/internal/handler/theme"
	searchhandler "blotting-consultancy/internal/handler/search"
	seoHandler "blotting-consultancy/internal/handler/seo"
	userHandler "blotting-consultancy/internal/handler/user"
	aiHandler "blotting-consultancy/internal/handler/ai"
	chunkedUploadHandler "blotting-consultancy/internal/handler/chunked_upload"
	mediaFolderHandler "blotting-consultancy/internal/handler/media_folder"
	migrationHandler "blotting-consultancy/internal/handler/migration"
	qaHandler "blotting-consultancy/internal/handler/qa"
	siteHandler "blotting-consultancy/internal/handler/site"
	storageHandler "blotting-consultancy/internal/handler/storage"
	systemHandler "blotting-consultancy/internal/handler/system"
	translationHandler "blotting-consultancy/internal/handler/translation"
	unifiedPageHandler "blotting-consultancy/internal/handler/unified_page"
	pageTemplateHandler "blotting-consultancy/internal/handler/page_template"
	themeExportHandler "blotting-consultancy/internal/handler/theme_export"
	"blotting-consultancy/internal/migration"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/seed"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/audit"
	"blotting-consultancy/pkg/config"
	appLogger "blotting-consultancy/pkg/logger"
	"blotting-consultancy/pkg/metrics"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

// @title           Impress CMS API
// @version         1.0
// @description     Bilingual CMS backend API for Impress (印迹). Supports content management, articles, pages, themes, media, and more.
// @host            localhost:8088
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}" for JWT authentication
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := appLogger.New(cfg.Env, map[string]interface{}{
		"service": "blotting-consultancy-api",
		"version": Version,
	})
	log.Info("Starting server",
		"env", cfg.Env,
		"port", cfg.Port,
		"version", Version,
		"buildTime", BuildTime,
		"gitCommit", GitCommit,
		"gitBranch", GitBranch,
	)

	// Initialize database
	logLevel := logger.Info
	if cfg.Env == "development" {
		logLevel = logger.Info
	} else if cfg.Env == "production" {
		logLevel = logger.Warn
	}

	maxOpenConn := 25
	maxIdleConn := 5
	maxLifetime := 5 * time.Minute
	if !db.IsPostgresDSN(cfg.DBDSN) {
		// SQLite is file-based and works best with a small connection pool.
		maxOpenConn = 1
		maxIdleConn = 1
		maxLifetime = 0
	}

	database, err := db.Init(db.InitOptions{
		DSN:         cfg.DBDSN,
		MaxOpenConn: maxOpenConn,
		MaxIdleConn: maxIdleConn,
		MaxLifetime: maxLifetime,
		LogLevel:    logLevel,
	})
	if err != nil {
		log.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	log.Info("Database connection established")

	// Fix legacy index conflict: SQLite indexes are database-global, and both
	// page_versions and content_versions had an index named idx_page_version.
	// Drop the stale one on page_versions so AutoMigrate can create idx_pv_page_version.
	database.DB.Exec("DROP INDEX IF EXISTS idx_page_version")

	// Run migrations
	migrator := db.NewMigrator(database)
	if err := migrator.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.ContentDocument{},
		// model.ContentVersion removed from AutoMigrate — table dropped by goose 00009
		&model.Media{},
		&model.PageView{},
		&model.Category{},
		&model.Tag{},
		&model.Article{},
		&model.BackupRecord{},
		&model.AuditEvent{},
		&model.Page{},
		&model.InstalledTheme{},
		&model.FormSubmission{},
		&model.MenuGroup{},
		&model.MenuItem{},
		&model.Comment{},
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
		&model.MarketplaceItem{},
		&model.MarketplaceVersion{},
		&model.MediaFolder{},
		&model.ChunkedUpload{},
		&model.QALog{},
		&model.Glossary{},
		&model.StorageConfig{},
		&model.Site{},
		&model.SiteUser{},
		&model.UnifiedPage{},
		&model.PageVersion{},
		&model.PageTemplate{},
		&model.SiteConfig{},
	); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Run data migrations
	if err := migrator.RunMigrations(db.DataMigrations()); err != nil {
		log.Error("Failed to run data migrations", "error", err)
		os.Exit(1)
	}
	// Run goose migrations (for schema changes beyond GORM AutoMigrate)
	{
		sqlDB, err := database.DB.DB()
		if err != nil {
			log.Error("Failed to get sql.DB for goose migrations", "error", err)
			os.Exit(1)
		}
		goose.SetBaseFS(migrations.EmbedMigrations)
		dialect := "sqlite3"
		if db.IsPostgresDSN(cfg.DBDSN) {
			dialect = "postgres"
		}
		if err := goose.SetDialect(dialect); err != nil {
			log.Error("Failed to set goose dialect", "error", err)
			os.Exit(1)
		}
		if err := goose.Up(sqlDB, "."); err != nil {
			log.Error("Failed to run goose migrations", "error", err)
			os.Exit(1)
		}
		log.Info("Goose migrations applied successfully")
	}
	log.Info("Database migrations completed")

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.HealthCheck(ctx); err != nil {
		log.Error("Database health check failed", "error", err)
		os.Exit(1)
	}
	log.Info("Database health check passed")

	// Initialize repositories
	userRepo := repository.NewGormUserRepository(database.DB)
	refreshTokenRepo := repository.NewGormRefreshTokenRepository(database.DB)
	contentDocRepo := repository.NewGormContentDocumentRepository(database.DB)
	// contentVersionRepo removed — old content version system replaced by page versions
	mediaRepo := repository.NewGormMediaRepository(database.DB)
	pageViewRepo := repository.NewGormPageViewRepository(database.DB)
	categoryRepo := repository.NewGormCategoryRepository(database.DB)
	tagRepo := repository.NewGormTagRepository(database.DB)
	articleRepo := repository.NewGormArticleRepository(database.DB)
	auditEventRepo := repository.NewGormAuditEventRepository(database.DB)
	pageRepo := repository.NewGormPageRepository(database.DB)
	installedThemeRepo := repository.NewGormInstalledThemeRepository(database.DB)
	formSubmissionRepo := repository.NewGormFormSubmissionRepository(database.DB)
	menuRepo := repository.NewGormMenuRepository(database.DB)
	commentRepo := repository.NewGormCommentRepository(database.DB)
	roleRepo := repository.NewGormRoleRepository(database.DB)
	marketplaceRepo := repository.NewGormMarketplaceRepository(database.DB)
	mediaFolderRepo := repository.NewGormMediaFolderRepository(database.DB)
	chunkedUploadRepo := repository.NewGormChunkedUploadRepository(database.DB)
	qaLogRepo := repository.NewGormQALogRepository(database.DB)
	glossaryRepo := repository.NewGormGlossaryRepository(database.DB)
	storageConfigRepo := repository.NewGormStorageConfigRepository(database.DB)
	siteRepo := repository.NewGormSiteRepository(database.DB)
	unifiedPageRepo := repository.NewGormUnifiedPageRepository(database.DB)
	pageVersionRepo := repository.NewGormPageVersionRepository(database.DB)
	pageTemplateRepo := repository.NewGormPageTemplateRepository(database.DB)
	siteConfigRepo := repository.NewGormSiteConfigRepository(database.DB)
	log.Info("Repositories initialized")

	// Initialize theme page service early (needed for seeding)
	themePageService := service.NewThemePageService(pageRepo)

	// Run seed (idempotent)
	seeder := seed.NewSeeder(userRepo, contentDocRepo, installedThemeRepo, themePageService, unifiedPageRepo, pageTemplateRepo)
	seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer seedCancel()
	if err := seeder.SeedAll(seedCtx); err != nil {
		log.Error("Failed to seed initial data", "error", err)
		os.Exit(1)
	}
	// Seed RBAC roles and permissions
	if err := seed.SeedRBAC(seedCtx, roleRepo); err != nil {
		log.Error("Failed to seed RBAC data", "error", err)
		os.Exit(1)
	}
	log.Info("Seed data initialized")

	// (old validationService + contentService removed — replaced by UnifiedPageService)
	log.Info("Services initialized")

	// Start scheduler for auto-publishing scheduled content
	schedulerService := service.NewSchedulerService(database.DB)
	schedulerService.Start()

	// Initialize audit logger (kept for future use)
	auditDbWriter := audit.NewDbWriter(auditEventRepo)
	_ = auditDbWriter // available for future use

	// Initialize backup service
	backupSvc := backup.NewService(database.DB, "./backups", 10, cfg.UploadDir, Version)
	log.Info("Audit logger and backup service initialized")

	// Initialize search service (needed by article handler)
	searchService := service.NewSearchService(database.DB, db.IsPostgresDSN(cfg.DBDSN))

	// Initialize chunked upload service
	chunkedUploadSvc := service.NewChunkedUploadService(
		chunkedUploadRepo,
		mediaRepo,
		"./tmp/uploads",
		cfg.UploadDir,
		"",
	)

	// Initialize site service
	siteSvc := service.NewSiteService(database.DB, siteRepo)

	// Initialize migration service
	migrationSvc := migration.NewService(articleRepo, categoryRepo, tagRepo)

	// Initialize translation provider (noop by default)
	translationProvider := service.NewNoopTranslationProvider()

	// Initialize event bus
	bus := eventbus.New()
	bus.Subscribe(eventbus.ContentCreated, eventbus.AsyncHandler(func(e eventbus.Event) {
		log.Info("Content event", "type", e.Type)
	}))
	bus.Subscribe(eventbus.ContentUpdated, eventbus.AsyncHandler(func(e eventbus.Event) {
		log.Info("Content event", "type", e.Type)
	}))
	bus.Subscribe(eventbus.ContentDeleted, eventbus.AsyncHandler(func(e eventbus.Event) {
		log.Info("Content event", "type", e.Type)
	}))
	log.Info("Event bus initialized")

	// Initialize provider registry with defaults
	registry := provider.NewRegistry()
	registry.Register("notifier", service.NewLogNotifier())
	registry.Register("captcha", &provider.NoopCaptchaProvider{})
	registry.Register("storage", service.NewLocalStorage(cfg.UploadDir))
	log.Info("Provider registry initialized", "providers", registry.List())

	// Initialize QA / embedding services (depend on registry.AI())
	vectorStore := service.NewMemoryVectorStore()
	qaService := service.NewQAService(registry.AI(), vectorStore)
	embeddingService := service.NewEmbeddingService(registry.AI(), vectorStore)

	// Initialize handlers
	authHandlerInst := authHandler.NewHandler(userRepo, refreshTokenRepo, cfg)
	// (old contentHandlerInst removed — replaced by unified page handler)
	publicHandlerInst := publicHandler.NewHandler(contentDocRepo, pageViewRepo, unifiedPageRepo)
	mediaHandlerInst := mediaHandler.NewHandler(mediaRepo, cfg.UploadDir, "")
	analyticsHandlerInst := analyticsHandler.NewHandler(pageViewRepo)
	categoryHandlerInst := categoryHandler.NewHandler(categoryRepo, articleRepo)
	tagHandlerInst := tagHandler.NewHandler(tagRepo, articleRepo)
	menuHandlerInst := menuHandler.NewHandler(menuRepo)
	articleHandlerInst := articleHandler.NewHandler(articleRepo, categoryRepo, tagRepo, searchService, bus)
	backupHandlerInst := backupHandler.NewHandler(backupSvc)
	auditlogHandlerInst := auditlogHandler.NewHandler(auditEventRepo)
	sitemapHandlerInst := sitemapHandler.NewHandler(contentDocRepo, articleRepo, cfg.BaseURL)
	// (old pageHandlerInst removed — replaced by unified page handler)
	themeHandlerInst := themeHandler.NewHandler(siteConfigRepo)
	installedThemeHandlerInst := installedThemeHandler.NewHandler(installedThemeRepo, themePageService)
	bootstrapHandlerInst := bootstrapHandler.NewHandler(contentDocRepo, installedThemeRepo, pageRepo)
	formSubmissionHandlerInst := formSubmissionHandler.NewHandler(formSubmissionRepo)
	userHandlerInst := userHandler.NewHandler(userRepo)
	seoHandlerInst := seoHandler.NewHandler(database.DB)
	captchaProvider := &provider.NoopCaptchaProvider{}
	antispamService := service.NewAntiSpamService(captchaProvider)
	commentHandlerInst := commentHandler.NewHandler(commentRepo, antispamService)
	searchHandlerInst := searchhandler.NewHandler(searchService)
	roleHandlerInst := roleHandler.NewHandler(roleRepo, userRepo)
	marketplaceSvc := service.NewMarketplaceService(marketplaceRepo)
	marketplaceHandlerInst := marketplaceHandler.NewHandler(marketplaceSvc)
	wizardSvc := service.NewWizardService(registry.AI(), pageRepo)
	wizardHandlerInst := wizardHandler.NewHandler(wizardSvc)
	aiHandlerInst := aiHandler.NewHandler(registry)
	chunkedUploadHandlerInst := chunkedUploadHandler.NewHandler(chunkedUploadSvc)
	mediaFolderHandlerInst := mediaFolderHandler.NewHandler(mediaFolderRepo, mediaRepo)
	migrationHandlerInst := migrationHandler.NewHandler(migrationSvc)
	qaHandlerInst := qaHandler.NewHandler(qaService, embeddingService, qaLogRepo, contentDocRepo, articleRepo)
	siteHandlerInst := siteHandler.NewHandler(siteSvc, siteRepo)
	storageHandlerInst := storageHandler.NewHandler(storageConfigRepo)
	systemHandlerInst := systemHandler.NewHandler(database.DB, cfg.UploadDir)
	translationHandlerInst := translationHandler.NewHandler(translationProvider, glossaryRepo, articleRepo)
	unifiedPageSvc := service.NewUnifiedPageService(unifiedPageRepo, pageVersionRepo)
	unifiedPageHdl := unifiedPageHandler.NewHandler(unifiedPageRepo, pageVersionRepo, unifiedPageSvc)
	pageTemplateHdl := pageTemplateHandler.NewHandler(pageTemplateRepo)
	themeExportSvc := service.NewThemeExportService(pageTemplateRepo, siteConfigRepo)
	themeExportHdl := themeExportHandler.NewHandler(themeExportSvc)
	log.Info("Handlers initialized")

	// Setup Gin router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Global middleware (order matters!)
	router.Use(gin.Recovery()) // Panic recovery
	router.Use(ginLogger(log)) // Request logging

	corsConfig := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "If-Match"},
		MaxAge:       10 * time.Minute,
	}
	if len(cfg.CORSAllowedOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.CORSAllowedOrigins
	} else {
		corsConfig.AllowAllOrigins = true
		log.Warn("CORS allowed origins not configured; falling back to allow all origins")
	}
	router.Use(cors.New(corsConfig))
	router.Use(apierror.ErrorHandler()) // API error handling

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
		if err := database.HealthCheck(ctx); err != nil {
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
	router.GET("/sitemap.xml", sitemapHandlerInst.GetSitemap)

	// Robots.txt (no auth required)
	router.GET("/robots.txt", seoHandlerInst.GetRobotsTxt)

	// Public routes (no auth required)
	publicGroup := router.Group("/public")
	publicGroup.Use(middleware.PublicRateLimit())
	{
		publicGroup.GET("/bootstrap", bootstrapHandlerInst.PublicBootstrap)
		publicGroup.GET("/content/:pageKey", publicHandlerInst.GetPublicContent)

		// Public article routes
		publicGroup.GET("/articles", articleHandlerInst.PublicList)
		publicGroup.GET("/articles/:slug", articleHandlerInst.PublicGetBySlug)

		// (old page routes removed — replaced by unified pages below)

		// Public category routes
		publicGroup.GET("/categories", categoryHandlerInst.PublicList)
		publicGroup.GET("/categories/:slug", categoryHandlerInst.PublicGetBySlug)

		// Public tag routes
		publicGroup.GET("/tags", tagHandlerInst.PublicList)
		publicGroup.GET("/tags/:slug", tagHandlerInst.PublicGetBySlug)

		// Public menu route
		publicGroup.GET("/menu", menuHandlerInst.PublicGetPrimary)

		// Public theme route
		publicGroup.GET("/theme", themeHandlerInst.PublicGet)

		// Public active theme route
		publicGroup.GET("/active-theme", installedThemeHandlerInst.PublicGetActive)

		// (old theme-pages route removed — theme pages are now part of unified pages)

		// Unified pages (replaces old page routes)
		publicGroup.GET("/pages", unifiedPageHdl.PublicList)
		publicGroup.GET("/pages/:slug", unifiedPageHdl.PublicGetBySlug)
	}

	// Public Q&A (knowledge base ask)
	publicGroup.POST("/qa/ask", qaHandlerInst.PublicAsk)

	// Form submission (public, with dedicated rate limit)
	router.POST("/public/form-submissions", middleware.FormSubmitRateLimit(), formSubmissionHandlerInst.HandlePublicSubmit)

	// Auth routes (no auth middleware, but handlers validate credentials)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", middleware.LoginRateLimit(), authHandlerInst.Login)
		authGroup.POST("/refresh", authHandlerInst.Refresh)
		authGroup.POST("/logout", authHandlerInst.Logout)

		// Protected auth routes
		authProtected := authGroup.Group("")
		authProtected.Use(middleware.Auth(cfg.JWTSecret))
		{
			authProtected.GET("/me", authHandlerInst.Me)
		}
	}

	// Initialize SEO renderer when FRONTEND_DIR is configured
	var seoRenderer *seo.Renderer
	if cfg.FrontendDir != "" {
		seoIndexPath := filepath.Join(cfg.FrontendDir, "index.html")
		var err error
		seoRenderer, err = seo.NewRenderer(seoIndexPath)
		if err != nil {
			log.Error("Failed to create SEO renderer", "error", err)
			os.Exit(1)
		}
		log.Info("SEO renderer initialized")
	}

	// Admin routes (require authentication and authorization)
	adminGroup := router.Group("/admin")
	// SPA fallback: if FRONTEND_DIR is set and the browser asks for HTML,
	// serve index.html instead of requiring auth (the SPA handles its own auth).
	if cfg.FrontendDir != "" {
		indexPath := filepath.Join(cfg.FrontendDir, "index.html")
		adminGroup.Use(func(c *gin.Context) {
			accept := c.GetHeader("Accept")
			if c.Request.Method == "GET" && strings.Contains(accept, "text/html") {
				if !serveSPAWithMeta(c, seoRenderer, cfg.BaseURL) {
					c.File(indexPath)
					c.Abort()
				}
				return
			}
			c.Next()
		})
	}
	adminGroup.Use(middleware.Auth(cfg.JWTSecret))
	// Legacy middleware kept for backward compatibility with existing JWT tokens.
	// New RBAC permission checks are applied at the route-group level.
	adminGroup.Use(middleware.RequireAdminOrEditor())
	{
		// (old content handler routes removed — replaced by unified page draft/publish/version routes)

		// Media management
		adminGroup.POST("/media/upload", mediaHandlerInst.Upload)
		adminGroup.GET("/media", mediaHandlerInst.List)
		adminGroup.DELETE("/media/:id", mediaHandlerInst.Delete)
		adminGroup.PUT("/media/:id/crop", mediaHandlerInst.Recrop)
		adminGroup.PUT("/media/:id", mediaHandlerInst.Rename)
		adminGroup.GET("/media/:id/usages", mediaHandlerInst.GetUsages)

		// Analytics (requires analytics:read via RBAC)
		adminAnalytics := adminGroup.Group("")
		adminAnalytics.Use(middleware.RequirePermission("analytics", "read", userRepo))
		{
			adminAnalytics.GET("/analytics/summary", analyticsHandlerInst.GetSummary)
		}

		// Article management
		adminGroup.GET("/articles", articleHandlerInst.AdminList)
		adminGroup.GET("/articles/:id", articleHandlerInst.AdminGetByID)
		adminGroup.POST("/articles", articleHandlerInst.AdminCreate)
		adminGroup.PUT("/articles/:id", articleHandlerInst.AdminUpdate)
		adminGroup.DELETE("/articles/:id", articleHandlerInst.AdminDelete)
		adminGroup.GET("/articles/:id/export", articleHandlerInst.AdminExportMarkdown)
		adminGroup.POST("/articles/import", articleHandlerInst.AdminImportMarkdown)

		// Category management
		adminGroup.GET("/categories", categoryHandlerInst.List)
		adminGroup.GET("/categories/tree", categoryHandlerInst.ListTree)
		adminGroup.GET("/categories/:id", categoryHandlerInst.GetByID)
		adminGroup.POST("/categories", categoryHandlerInst.Create)
		adminGroup.PUT("/categories/:id", categoryHandlerInst.Update)
		adminGroup.DELETE("/categories/:id", categoryHandlerInst.Delete)

		// Tag management
		adminGroup.GET("/tags", tagHandlerInst.List)
		adminGroup.POST("/tags", tagHandlerInst.Create)
		adminGroup.PUT("/tags/:id", tagHandlerInst.Update)
		adminGroup.DELETE("/tags/:id", tagHandlerInst.Delete)

		// Menu management
		adminGroup.GET("/menus", menuHandlerInst.ListGroups)
		adminGroup.POST("/menus", menuHandlerInst.CreateGroup)
		adminGroup.GET("/menus/:id", menuHandlerInst.GetGroup)
		adminGroup.PUT("/menus/:id", menuHandlerInst.UpdateGroup)
		adminGroup.DELETE("/menus/:id", menuHandlerInst.DeleteGroup)
		adminGroup.PUT("/menus/:id/primary", menuHandlerInst.SetPrimary)
		adminGroup.POST("/menus/:id/items", menuHandlerInst.CreateItem)
		adminGroup.PUT("/menus/:id/items/:itemId", menuHandlerInst.UpdateItem)
		adminGroup.DELETE("/menus/:id/items/:itemId", menuHandlerInst.DeleteItem)
		adminGroup.PUT("/menus/:id/items/reorder", menuHandlerInst.ReorderItems)

		// Backup management
		adminGroup.GET("/backups", backupHandlerInst.List)
		adminGroup.POST("/backups/trigger", backupHandlerInst.Trigger)

		// Site export/import (requires backups:manage via RBAC)
		adminBackup := adminGroup.Group("/backups")
		adminBackup.Use(middleware.RequirePermission("backups", "manage", userRepo))
		{
			adminBackup.POST("/export", backupHandlerInst.Export)
			adminBackup.GET("/export/:filename", backupHandlerInst.DownloadExport)
			adminBackup.POST("/import", backupHandlerInst.Import)
			adminBackup.POST("/import/validate", backupHandlerInst.ValidateImport)
		}

		// Audit logs (requires audit_logs:read via RBAC)
		adminAudit := adminGroup.Group("")
		adminAudit.Use(middleware.RequirePermission("audit_logs", "read", userRepo))
		{
			adminAudit.GET("/audit-logs", auditlogHandlerInst.List)
		}

		// (old page management routes removed — replaced by unified page routes below)

		// Theme token management (existing)
		adminGroup.GET("/theme", themeHandlerInst.AdminGet)
		adminGroup.PUT("/theme", themeHandlerInst.AdminUpdate)

		// Installed theme management
		adminGroup.GET("/themes", installedThemeHandlerInst.AdminList)
		adminGroup.GET("/themes/:id", installedThemeHandlerInst.AdminGetByID)
		adminGroup.POST("/themes", installedThemeHandlerInst.AdminCreate)
		adminGroup.PUT("/themes/:id", installedThemeHandlerInst.AdminUpdate)
		adminGroup.DELETE("/themes/:id", installedThemeHandlerInst.AdminDelete)
		adminGroup.PUT("/themes/:id/activate", installedThemeHandlerInst.AdminActivate)

		// Form submission management
		adminGroup.GET("/form-submissions/counts", formSubmissionHandlerInst.HandleAdminCounts)
		adminGroup.GET("/form-submissions", formSubmissionHandlerInst.HandleAdminList)
		adminGroup.GET("/form-submissions/:id", formSubmissionHandlerInst.HandleAdminGetByID)
		adminGroup.PATCH("/form-submissions/:id/status", formSubmissionHandlerInst.HandleAdminUpdateStatus)
		adminGroup.POST("/form-submissions/bulk-status", formSubmissionHandlerInst.HandleAdminBulkUpdateStatus)
		adminGroup.DELETE("/form-submissions/:id", formSubmissionHandlerInst.HandleAdminDelete)

		// User management (requires users:manage via RBAC)
		adminUsers := adminGroup.Group("/users")
		adminUsers.Use(middleware.RequirePermission("users", "manage", userRepo))
		{
			adminUsers.GET("", userHandlerInst.List)
			adminUsers.GET("/:id", userHandlerInst.GetByID)
			adminUsers.POST("", userHandlerInst.Create)
			adminUsers.PUT("/:id", userHandlerInst.Update)
			adminUsers.DELETE("/:id", userHandlerInst.Delete)
		}

		// RBAC Role management (requires roles:manage via RBAC)
		adminRoles := adminGroup.Group("/roles")
		adminRoles.Use(middleware.RequirePermission("roles", "manage", userRepo))
		{
			adminRoles.GET("", roleHandlerInst.List)
			adminRoles.GET("/:id", roleHandlerInst.GetByID)
			adminRoles.POST("", roleHandlerInst.Create)
			adminRoles.PUT("/:id", roleHandlerInst.Update)
			adminRoles.DELETE("/:id", roleHandlerInst.Delete)
			adminRoles.POST("/assign", roleHandlerInst.AssignRole)
			adminRoles.POST("/unassign", roleHandlerInst.UnassignRole)
			adminRoles.GET("/user/:userId", roleHandlerInst.GetUserRoles)
		}

		// Permission listing (requires roles:read via RBAC)
		adminGroup.GET("/permissions", roleHandlerInst.ListPermissions)

		// AI Site Building Wizard
		adminGroup.POST("/wizard/generate-plan", wizardHandlerInst.GeneratePlan)
		adminGroup.POST("/wizard/apply-plan", wizardHandlerInst.ApplyPlan)
		adminGroup.POST("/wizard/suggest-colors", wizardHandlerInst.SuggestColors)
		adminGroup.POST("/wizard/generate-content", wizardHandlerInst.GenerateContent)

		// Marketplace (plugin/theme registry)
		adminGroup.GET("/marketplace/items", marketplaceHandlerInst.AdminListItems)
		adminGroup.GET("/marketplace/installed", marketplaceHandlerInst.AdminListInstalled)
		adminGroup.POST("/marketplace/items", marketplaceHandlerInst.AdminRegisterItem)
		adminGroup.GET("/marketplace/items/:slug", marketplaceHandlerInst.AdminGetItem)
		adminGroup.POST("/marketplace/items/:slug/install", marketplaceHandlerInst.AdminInstallItem)
		adminGroup.PUT("/marketplace/items/:slug/update", marketplaceHandlerInst.AdminUpdateItem)
		adminGroup.DELETE("/marketplace/items/:slug", marketplaceHandlerInst.AdminUninstallItem)
		adminGroup.POST("/marketplace/items/:slug/versions", marketplaceHandlerInst.AdminAddVersion)

		// AI provider management
		adminGroup.POST("/ai/chat", aiHandlerInst.Chat)
		adminGroup.POST("/ai/summarize", aiHandlerInst.Summarize)
		adminGroup.POST("/ai/suggest-titles", aiHandlerInst.SuggestTitles)
		adminGroup.POST("/ai/suggest-tags", aiHandlerInst.SuggestTags)
		adminGroup.POST("/ai/complete", aiHandlerInst.Complete)
		adminGroup.GET("/ai/config", aiHandlerInst.GetConfig)
		adminGroup.PUT("/ai/config", aiHandlerInst.UpdateConfig)

		// Chunked upload
		adminGroup.POST("/media/upload/init", chunkedUploadHandlerInst.InitUpload)
		adminGroup.POST("/media/upload/:uploadId/chunk", chunkedUploadHandlerInst.UploadChunk)
		adminGroup.POST("/media/upload/:uploadId/complete", chunkedUploadHandlerInst.CompleteUpload)

		// Media folders
		adminGroup.GET("/media/folders", mediaFolderHandlerInst.ListTree)
		adminGroup.POST("/media/folders", mediaFolderHandlerInst.Create)
		adminGroup.PUT("/media/folders/:id", mediaFolderHandlerInst.Rename)
		adminGroup.DELETE("/media/folders/:id", mediaFolderHandlerInst.Delete)
		adminGroup.PUT("/media/:id/move", mediaFolderHandlerInst.MoveMedia)

		// Data migration (import from WordPress, Halo, Markdown)
		adminGroup.POST("/migration/import", migrationHandlerInst.Import)
		adminGroup.GET("/migration/jobs", migrationHandlerInst.ListJobs)
		adminGroup.GET("/migration/jobs/:jobId", migrationHandlerInst.GetJob)
		adminGroup.GET("/migration/jobs/:jobId/stream", migrationHandlerInst.StreamProgress)

		// Knowledge base Q&A (admin)
		adminGroup.POST("/qa/index", qaHandlerInst.AdminIndex)
		adminGroup.GET("/qa/logs", qaHandlerInst.AdminListLogs)
		adminGroup.POST("/qa/logs/:id/feedback", qaHandlerInst.AdminFeedback)

		// Site management
		adminGroup.GET("/sites", siteHandlerInst.AdminList)
		adminGroup.GET("/sites/:id", siteHandlerInst.AdminGetByID)
		adminGroup.POST("/sites", siteHandlerInst.AdminCreate)
		adminGroup.PUT("/sites/:id", siteHandlerInst.AdminUpdate)
		adminGroup.DELETE("/sites/:id", siteHandlerInst.AdminDelete)
		adminGroup.GET("/sites/:id/users", siteHandlerInst.AdminListUsers)
		adminGroup.POST("/sites/:id/users", siteHandlerInst.AdminAssignUser)
		adminGroup.DELETE("/sites/:id/users/:userId", siteHandlerInst.AdminUnassignUser)
		adminGroup.GET("/sites/:id/export", siteHandlerInst.AdminExport)
		adminGroup.POST("/sites/import", siteHandlerInst.AdminImport)

		// Storage configuration
		adminGroup.GET("/storage/config", storageHandlerInst.GetConfig)
		adminGroup.PUT("/storage/config", storageHandlerInst.UpdateConfig)
		adminGroup.POST("/storage/test", storageHandlerInst.TestConnection)

		// System status
		adminGroup.GET("/system/status", systemHandlerInst.GetStatus)

		// Translation & glossary
		adminGroup.POST("/translate", translationHandlerInst.Translate)
		adminGroup.POST("/translate/batch", translationHandlerInst.BatchTranslate)
		adminGroup.POST("/translate/article/:id", translationHandlerInst.TranslateArticle)
		adminGroup.GET("/glossary", translationHandlerInst.GlossaryList)
		adminGroup.POST("/glossary", translationHandlerInst.GlossaryCreate)
		adminGroup.PUT("/glossary/:id", translationHandlerInst.GlossaryUpdate)
		adminGroup.DELETE("/glossary/:id", translationHandlerInst.GlossaryDelete)

		// Page management (unified page system)
		adminGroup.GET("/pages", unifiedPageHdl.AdminList)
		adminGroup.GET("/pages/:id", unifiedPageHdl.AdminGetByID)
		adminGroup.POST("/pages", unifiedPageHdl.AdminCreate)
		adminGroup.GET("/pages/:id/draft", unifiedPageHdl.AdminGetDraft)
		adminGroup.PUT("/pages/:id/draft", unifiedPageHdl.AdminUpdateDraft)
		adminGroup.POST("/pages/:id/publish", unifiedPageHdl.AdminPublish)
		adminGroup.POST("/pages/:id/unpublish", unifiedPageHdl.AdminUnpublish)
		adminGroup.POST("/pages/:id/rollback", unifiedPageHdl.AdminRollback)
		adminGroup.GET("/pages/:id/versions", unifiedPageHdl.AdminListVersions)
		adminGroup.GET("/pages/:id/versions/:version", unifiedPageHdl.AdminGetVersionDetail)
		adminGroup.DELETE("/pages/:id", unifiedPageHdl.AdminDelete)

		// Page template management
		adminGroup.GET("/templates", pageTemplateHdl.List)
		adminGroup.POST("/templates", pageTemplateHdl.Create)
		adminGroup.PUT("/templates/:id", pageTemplateHdl.Update)
		adminGroup.DELETE("/templates/:id", pageTemplateHdl.Delete)
		adminGroup.POST("/templates/:id/duplicate", pageTemplateHdl.Duplicate)

		// Theme export/import
		adminGroup.POST("/theme-packages/export", themeExportHdl.Export)
		adminGroup.POST("/theme-packages/import", themeExportHdl.Import)
		adminGroup.GET("/theme-packages", themeExportHdl.List)
		adminGroup.PUT("/theme-packages/:id/apply", themeExportHdl.Apply)
	}

	// SEO routes (public + admin)
	seoHandlerInst.RegisterRoutes(publicGroup, adminGroup)

	// Comment routes (public + admin)
	commentHandlerInst.RegisterRoutes(publicGroup, adminGroup)

	// Search routes (public + admin)
	searchHandlerInst.RegisterRoutes(publicGroup, adminGroup)

	// Serve uploaded files statically
	router.Static("/uploads", cfg.UploadDir)

	// Serve frontend static assets when FRONTEND_DIR is configured
	if cfg.FrontendDir != "" {
		router.Static("/assets", filepath.Join(cfg.FrontendDir, "assets"))
		router.Static("/images", filepath.Join(cfg.FrontendDir, "images"))
		router.StaticFile("/favicon.ico", filepath.Join(cfg.FrontendDir, "favicon.ico"))

		// SPA fallback: non-API GET requests return index.html with SEO meta
		indexHTML := filepath.Join(cfg.FrontendDir, "index.html")
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if c.Request.Method == "GET" &&
				!strings.HasPrefix(path, "/public/") &&
				!strings.HasPrefix(path, "/auth/") &&
				!strings.HasPrefix(path, "/uploads/") &&
				path != "/health" &&
				path != "/version" &&
				path != "/metrics" &&
				path != "/sitemap.xml" &&
				path != "/robots.txt" {
				if !serveSPAWithMeta(c, seoRenderer, cfg.BaseURL) {
					http.ServeFile(c.Writer, c.Request, indexHTML)
					c.Abort()
				}
				return
			}
			c.JSON(404, gin.H{"error": "not found"})
		})
	}

	log.Info("Router configured with all routes")

	// Setup HTTP server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("Server listening", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Server shutting down gracefully...")

	// Stop background services
	schedulerService.Stop()
	antispamService.Stop()

	// Shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	// Close database connection
	if err := database.Close(); err != nil {
		log.Error("Failed to close database connection", "error", err)
	}

	log.Info("Server stopped")
}

// serveSPAWithMeta renders index.html with SEO meta tags. Returns true if served
// successfully, false if caller should fall back to static file serving.
func serveSPAWithMeta(c *gin.Context, renderer *seo.Renderer, baseURL string) bool {
	if renderer == nil {
		return false
	}
	locale := c.DefaultQuery("locale", "zh")
	if locale != "zh" && locale != "en" {
		locale = "zh"
	}
	meta := seo.ResolveFromPath(c.Request.URL.Path, baseURL, locale)
	html, err := renderer.Render(meta)
	if err != nil {
		return false
	}
	c.Data(200, "text/html; charset=utf-8", []byte(html))
	c.Abort()
	return true
}

// ginLogger returns a Gin middleware that logs requests using the app logger
func ginLogger(log *appLogger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		log.Info("Request",
			"method", method,
			"path", path,
			"status", status,
			"duration", duration.String(),
			"ip", c.ClientIP(),
		)
	}
}
