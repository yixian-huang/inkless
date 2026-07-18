package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "blotting-consultancy/docs/swagger" // swagger docs
	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/db/migrations"
	"blotting-consultancy/internal/eventbus"
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
	schedulerHandler "blotting-consultancy/internal/handler/scheduler"
	searchhandler "blotting-consultancy/internal/handler/search"
	seoHandler "blotting-consultancy/internal/handler/seo"
	setupHandler "blotting-consultancy/internal/handler/setup"
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
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/migration"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/module"
	backupMod "blotting-consultancy/internal/modules/backup"
	commentMod "blotting-consultancy/internal/modules/comment"
	formSubmissionMod "blotting-consultancy/internal/modules/form_submission"
	qa "blotting-consultancy/internal/modules/qa"
	pluginruntime "blotting-consultancy/internal/plugin"
	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/seed"
	"blotting-consultancy/internal/seo"
	"blotting-consultancy/internal/service"
	install "blotting-consultancy/internal/setup"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/audit"
	"blotting-consultancy/pkg/config"
	appLogger "blotting-consultancy/pkg/logger"
	"blotting-consultancy/pkg/secretcipher"
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
// @description     Bilingual CMS backend API for Impress. Supports content management, articles, pages, themes, media, and more.
// @host            localhost:8088
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}" for JWT authentication
func main() {
	// Load configuration (bootstrap mode allows missing JWT for first-run setup)
	loadResult, err := config.LoadWithBootstrap()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	cfg := loadResult.Config

	// Initialize logger
	log := appLogger.New(cfg.Env, map[string]interface{}{
		"service": "impress-api",
		"version": Version,
	})
	log.Info("Starting server",
		"env", cfg.Env,
		"port", cfg.Port,
		"version", Version,
		"buildTime", BuildTime,
		"gitCommit", GitCommit,
		"gitBranch", GitBranch,
		"bootstrapMode", loadResult.BootstrapMode,
	)
	if loadResult.BootstrapMode {
		log.Warn("Setup bootstrap mode active — use /setup to persist .env and restart")
	}

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
		maxOpenConn = 4
		maxIdleConn = 2
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
		&model.MenuGroup{},
		&model.MenuItem{},
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
		&model.MarketplaceItem{},
		&model.MarketplaceVersion{},
		&model.MediaFolder{},
		&model.ChunkedUpload{},
		&model.Glossary{},
		&model.StorageConfig{},
		&model.AIConfig{},
		&model.UnifiedPage{},
		&model.PageVersion{},
		&model.ScheduledPublishJob{},
		&model.PageTemplate{},
		&model.SiteConfig{},
		&model.Plugin{},
		&model.PluginSetting{},
		&commentMod.Comment{},
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
		migrations.Dialect = dialect
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
	menuRepo := repository.NewGormMenuRepository(database.DB)
	roleRepo := repository.NewGormRoleRepository(database.DB)
	marketplaceRepo := repository.NewGormMarketplaceRepository(database.DB)
	mediaFolderRepo := repository.NewGormMediaFolderRepository(database.DB)
	chunkedUploadRepo := repository.NewGormChunkedUploadRepository(database.DB)
	glossaryRepo := repository.NewGormGlossaryRepository(database.DB)
	storageConfigRepo := repository.NewGormStorageConfigRepository(database.DB)
	unifiedPageRepo := repository.NewGormUnifiedPageRepository(database.DB)
	pageVersionRepo := repository.NewGormPageVersionRepository(database.DB)
	scheduledPublishJobRepo := repository.NewGormScheduledPublishJobRepository(database.DB)
	pageTemplateRepo := repository.NewGormPageTemplateRepository(database.DB)
	siteConfigRepo := repository.NewGormSiteConfigRepository(database.DB)
	log.Info("Repositories initialized")

	// Initialize theme page service early (needed for seeding)
	themePageService := service.NewThemePageService(pageRepo)

	// Run seed (idempotent)
	seeder := seed.NewSeeder(userRepo, contentDocRepo, installedThemeRepo, themePageService, unifiedPageRepo, pageTemplateRepo, siteConfigRepo)
	seedRBAC := func(ctx context.Context, roleRepo repository.RoleRepository) error {
		return seed.SeedRBAC(ctx, roleRepo)
	}
	seederFactory := func(tx *gorm.DB) *seed.Seeder {
		txPageRepo := repository.NewGormPageRepository(tx)
		return seed.NewSeeder(
			repository.NewGormUserRepository(tx),
			repository.NewGormContentDocumentRepository(tx),
			repository.NewGormInstalledThemeRepository(tx),
			service.NewThemePageService(txPageRepo),
			repository.NewGormUnifiedPageRepository(tx),
			repository.NewGormPageTemplateRepository(tx),
			repository.NewGormSiteConfigRepository(tx),
		)
	}
	setupSvc := install.NewService(
		database.DB,
		userRepo,
		siteConfigRepo,
		seederFactory,
		seedRBAC,
		install.ServiceOptions{
			BootstrapMode:    loadResult.BootstrapMode,
			EnvSecretsLoaded: loadResult.EnvSecretsLoaded,
			DatabaseType:     config.DatabaseTypeFromDSN(cfg.DBDSN),
			EnvFilePath:      config.DefaultEnvFilePath(),
			ServerPort:       cfg.Port,
		},
	)
	seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer seedCancel()

	installed, err := setupSvc.IsInstalled(seedCtx)
	if err != nil {
		log.Error("Failed to check install status", "error", err)
		os.Exit(1)
	}

	seedMode := os.Getenv("SEED_MODE")
	if err := seed.RunStartupSeed(seedCtx, installed, seedMode, seeder, func(ctx context.Context) error {
		return seedRBAC(ctx, roleRepo)
	}); err != nil {
		log.Error("Failed to run startup seed", "error", err)
		os.Exit(1)
	}
	log.Info("Seed data initialized", "installed", installed, "seedMode", seedMode)

	// (old validationService + contentService removed — replaced by UnifiedPageService)
	log.Info("Services initialized")

	// Initialize database-backed audit writer. Audit failures are best-effort
	// for the request but are emitted to the application logger.
	auditDbWriter := audit.NewDbWriter(auditEventRepo, log)

	log.Info("Audit logger initialized")

	// Initialize search service (needed by article handler)
	searchService := service.NewSearchService(database.DB, db.IsPostgresDSN(cfg.DBDSN))

	// Initialize runtime providers and restore persisted configuration before
	// modules or handlers capture their dependencies.
	registry := provider.NewRegistry()
	registry.Register("notifier", service.NewLogNotifier())
	registry.Register("captcha", &provider.NoopCaptchaProvider{})

	secretCipher, err := secretcipher.New(cfg.JWTSecret)
	if err != nil {
		log.Error("Failed to initialize secret cipher", "error", err)
		os.Exit(1)
	}

	aiConfigSvc := service.NewAIConfigService(
		repository.NewGormAIConfigRepository(database.DB),
		secretCipher,
		registry,
	)
	if err := aiConfigSvc.Restore(context.Background()); err != nil {
		log.Error("Failed to restore AI configuration", "error", err)
		os.Exit(1)
	}

	storageRuntime := service.NewStorageRuntimeService(
		storageConfigRepo,
		registry,
		service.NewLocalStorage(cfg.UploadDir),
		secretCipher,
	)
	if err := storageRuntime.RestoreStartupConfig(context.Background()); err != nil {
		log.Error("Failed to restore storage configuration", "error", err)
		os.Exit(1)
	}

	pluginStore := pluginruntime.NewStore(database.DB)
	pluginManager := pluginruntime.NewManager(pluginruntime.ManagerConfig{
		PluginDir: cfg.PluginDir,
		DataDir:   cfg.PluginDataDir,
	}, pluginStore, registry)
	if cfg.ExternalPlugins {
		if err := pluginManager.StartEnabledPlugins(context.Background()); err != nil {
			log.Error("Failed to restore enabled plugins", "error", err)
			os.Exit(1)
		}
		pluginManager.StartHealthMonitor(30 * time.Second)
	}
	log.Info("Provider registry initialized", "providers", registry.List())

	// Initialize chunked upload service
	chunkedUploadSvc := service.NewChunkedUploadServiceWithStorage(
		chunkedUploadRepo,
		mediaRepo,
		"./tmp/uploads",
		cfg.UploadDir,
		"",
		storageRuntime,
	)

	// Initialize migration service
	migrationSvc := migration.NewService(articleRepo, categoryRepo, tagRepo)

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

	// Initialize in-memory TTL caches
	publicCache := cache.New(60 * time.Second)
	rbacCache := cache.New(30 * time.Second)

	// Initialize feature modules
	mgr := module.NewManager()
	commentModule := commentMod.New()
	mgr.Register(qa.New())
	mgr.Register(commentModule)
	mgr.Register(formSubmissionMod.New())
	backupModule := backupMod.New()
	mgr.Register(backupModule)
	if err := mgr.InitAll(module.Dependencies{
		DB:       database.DB,
		Registry: registry,
		Repos: &module.SharedRepos{
			ContentDoc: contentDocRepo,
			Article:    articleRepo,
		},
		SiteCfg:    siteConfigRepo,
		UserRepo:   userRepo,
		RBACCache:  rbacCache,
		UploadDir:  cfg.UploadDir,
		BackupDir:  cfg.BackupDir,
		AppVersion: Version,
	}); err != nil {
		log.Error("Failed to initialize modules", "error", err)
		os.Exit(1)
	}

	// Cache invalidation on content changes
	bus.Subscribe(eventbus.ContentCreated, eventbus.AsyncHandler(func(e eventbus.Event) {
		publicCache.DeletePrefix("articles:")
		publicCache.DeletePrefix("article:")
		publicCache.Flush() // bootstrap includes article data indirectly
	}))
	bus.Subscribe(eventbus.ContentUpdated, eventbus.AsyncHandler(func(e eventbus.Event) {
		publicCache.Flush() // simplest: flush all on any content change
	}))
	bus.Subscribe(eventbus.ContentDeleted, eventbus.AsyncHandler(func(e eventbus.Event) {
		publicCache.Flush()
	}))
	bus.Subscribe(eventbus.ContentPublished, eventbus.AsyncHandler(func(e eventbus.Event) {
		publicCache.Flush()
	}))

	// Initialize handlers
	authHandlerInst := authHandler.NewHandler(userRepo, refreshTokenRepo, cfg)
	publicHandlerInst := publicHandler.NewHandler(contentDocRepo, pageViewRepo, unifiedPageRepo, publicCache)
	mediaHandlerInst := mediaHandler.NewHandlerWithStorage(mediaRepo, cfg.UploadDir, "", storageRuntime)
	analyticsHandlerInst := analyticsHandler.NewHandler(pageViewRepo)
	categoryHandlerInst := categoryHandler.NewHandler(categoryRepo, articleRepo)
	tagHandlerInst := tagHandler.NewHandler(tagRepo, articleRepo)
	menuHandlerInst := menuHandler.NewHandler(menuRepo)
	articleHandlerInst := articleHandler.NewHandler(articleRepo, categoryRepo, tagRepo, searchService, bus, publicCache)
	auditlogHandlerInst := auditlogHandler.NewHandler(auditEventRepo)
	sitemapHandlerInst := sitemapHandler.NewHandler(contentDocRepo, articleRepo, cfg.BaseURL)
	feedHandlerInst := feedHandler.NewHandler(articleRepo, siteConfigRepo, cfg.BaseURL, "Blog", "Latest posts")
	themeHandlerInst := themeHandler.NewHandler(siteConfigRepo, publicCache)
	installedThemeHandlerInst := installedThemeHandler.NewHandler(installedThemeRepo, themePageService, publicCache)
	bootstrapHandlerInst := bootstrapHandler.NewHandler(contentDocRepo, installedThemeRepo, pageRepo, unifiedPageRepo, siteConfigRepo, publicCache)
	globalConfigHandlerInst := globalConfigHandler.NewHandler(contentDocRepo, publicCache)
	featuresHandlerInst := featuresHandler.NewHandler(siteConfigRepo, publicCache)
	emailSvc := service.NewEmailService(siteConfigRepo)
	emailSettingsHandlerInst := emailSettingsHandler.NewHandler(siteConfigRepo, emailSvc)
	userHandlerInst := userHandler.NewHandler(userRepo)
	seoHandlerInst := seoHandler.NewHandler(database.DB)
	searchHandlerInst := searchhandler.NewHandler(searchService)
	roleHandlerInst := roleHandler.NewHandler(roleRepo, userRepo)
	marketplaceSvc := service.NewMarketplaceService(marketplaceRepo)
	marketplaceHandlerInst := marketplaceHandler.NewHandler(marketplaceSvc)
	pluginHandlerInst := pluginHandler.NewHandler(pluginManager, registry, cfg.ExternalPlugins)
	wizardSvc := service.NewWizardServiceWithRegistry(registry, unifiedPageRepo)
	wizardHandlerInst := wizardHandler.NewHandler(wizardSvc)
	aiHandlerInst := aiHandler.NewHandler(registry, aiConfigSvc)
	chunkedUploadHandlerInst := chunkedUploadHandler.NewHandler(chunkedUploadSvc)
	mediaFolderHandlerInst := mediaFolderHandler.NewHandler(mediaFolderRepo, mediaRepo)
	migrationHandlerInst := migrationHandler.NewHandler(migrationSvc)
	storageHandlerInst := storageHandler.NewHandlerWithRuntime(storageRuntime)
	systemHandlerInst := systemHandler.NewHandler(database.DB, cfg.UploadDir, Version)
	translationHandlerInst := translationHandler.NewHandlerWithRegistry(registry, glossaryRepo, articleRepo)
	unifiedPageSvc := service.NewUnifiedPageService(unifiedPageRepo, pageVersionRepo, bus).
		WithAuditWriter(auditDbWriter)
	unifiedPageHdl := unifiedPageHandler.NewHandler(unifiedPageRepo, pageVersionRepo, unifiedPageSvc, publicCache, bus)
	articlePublicationSvc := service.NewArticlePublicationService(articleRepo, searchService, bus).
		WithTaxonomyRepositories(categoryRepo, tagRepo).
		WithAuditWriter(auditDbWriter)
	schedulerService := service.NewSchedulerService(scheduledPublishJobRepo, articlePublicationSvc, unifiedPageSvc)
	schedulerService.Start()
	schedulerHdl := schedulerHandler.NewHandler(schedulerService)
	pageTemplateHdl := pageTemplateHandler.NewHandler(pageTemplateRepo)
	themeExportSvc := service.NewThemeExportService(pageTemplateRepo, siteConfigRepo)
	themeExportHdl := themeExportHandler.NewHandler(themeExportSvc)
	setupHandlerInst := setupHandler.NewHandler(setupSvc)
	log.Info("Handlers initialized")

	// Setup Gin router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Global middleware (order matters!)
	router.Use(gin.Recovery())                     // Panic recovery
	router.Use(ginLogger(log))                     // Request logging
	router.Use(gzip.Gzip(gzip.DefaultCompression)) // Gzip compression
	router.Use(middleware.AuditContext())          // Request metadata for audit events

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

	// Register all routes
	handlers := &Handlers{
		Auth:           authHandlerInst,
		Article:        articleHandlerInst,
		Public:         publicHandlerInst,
		Bootstrap:      bootstrapHandlerInst,
		Media:          mediaHandlerInst,
		Analytics:      analyticsHandlerInst,
		Category:       categoryHandlerInst,
		Tag:            tagHandlerInst,
		Menu:           menuHandlerInst,
		AuditLog:       auditlogHandlerInst,
		Sitemap:        sitemapHandlerInst,
		Feed:           feedHandlerInst,
		Theme:          themeHandlerInst,
		InstalledTheme: installedThemeHandlerInst,
		EmailSettings:  emailSettingsHandlerInst,
		Features:       featuresHandlerInst,
		GlobalConfig:   globalConfigHandlerInst,
		User:           userHandlerInst,
		SEO:            seoHandlerInst,
		Search:         searchHandlerInst,
		Role:           roleHandlerInst,
		Marketplace:    marketplaceHandlerInst,
		Plugin:         pluginHandlerInst,
		Wizard:         wizardHandlerInst,
		AI:             aiHandlerInst,
		ChunkedUpload:  chunkedUploadHandlerInst,
		MediaFolder:    mediaFolderHandlerInst,
		Migration:      migrationHandlerInst,
		Storage:        storageHandlerInst,
		System:         systemHandlerInst,
		Translation:    translationHandlerInst,
		UnifiedPage:    unifiedPageHdl,
		Scheduler:      schedulerHdl,
		PageTemplate:   pageTemplateHdl,
		ThemeExport:    themeExportHdl,
	}
	routeDeps := &RouteDeps{
		UserRepo:       userRepo,
		RBACCache:      rbacCache,
		Cfg:            cfg,
		Database:       database,
		ModuleMgr:      mgr,
		ContentDocRepo: contentDocRepo,
		AuditWriter:    auditDbWriter,
	}
	registerRoutes(router, handlers, routeDeps)
	setupHandlerInst.RegisterRoutes(router, middleware.LoginRateLimit())

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
	commentModule.Stop()

	// Shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}
	if err := pluginManager.StopAll(); err != nil {
		log.Error("Failed to stop plugins cleanly", "error", err)
	}

	// Close database connection
	if err := database.Close(); err != nil {
		log.Error("Failed to close database connection", "error", err)
	}

	log.Info("Server stopped")
}

// serveSPAWithMeta renders index.html with SEO meta tags. Returns true if served
// successfully, false if caller should fall back to static file serving.
func serveSPAWithMeta(c *gin.Context, renderer *seo.Renderer, baseURL string, contentDocRepo repository.ContentDocumentRepository) bool {
	if renderer == nil {
		return false
	}
	locale := c.DefaultQuery("locale", "zh")
	if locale != "zh" && locale != "en" {
		locale = "zh"
	}
	meta := seo.ResolveFromPath(c.Request.URL.Path, baseURL, locale)
	if contentDocRepo != nil {
		if doc, err := contentDocRepo.FindByPageKey(c.Request.Context(), model.PageKeyGlobal); err == nil && doc != nil {
			meta.ApplyGlobal(map[string]any(doc.PublishedConfig), locale)
		}
	}
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
