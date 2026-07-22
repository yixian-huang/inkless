package app

import (
	"context"
	"fmt"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/db"
	"github.com/yixian-huang/inkless/backend/internal/eventbus"
	aiHandler "github.com/yixian-huang/inkless/backend/internal/handler/ai"
	apiKeyHandler "github.com/yixian-huang/inkless/backend/internal/handler/api_key"
	analyticsHandler "github.com/yixian-huang/inkless/backend/internal/handler/analytics"
	articleHandler "github.com/yixian-huang/inkless/backend/internal/handler/article"
	auditlogHandler "github.com/yixian-huang/inkless/backend/internal/handler/auditlog"
	authHandler "github.com/yixian-huang/inkless/backend/internal/handler/auth"
	bootstrapHandler "github.com/yixian-huang/inkless/backend/internal/handler/bootstrap"
	categoryHandler "github.com/yixian-huang/inkless/backend/internal/handler/category"
	chunkedUploadHandler "github.com/yixian-huang/inkless/backend/internal/handler/chunked_upload"
	dashboardHandler "github.com/yixian-huang/inkless/backend/internal/handler/dashboard"
	emailSettingsHandler "github.com/yixian-huang/inkless/backend/internal/handler/email_settings"
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
	schedulerHandler "github.com/yixian-huang/inkless/backend/internal/handler/scheduler"
	searchhandler "github.com/yixian-huang/inkless/backend/internal/handler/search"
	seoHandler "github.com/yixian-huang/inkless/backend/internal/handler/seo"
	setupHandler "github.com/yixian-huang/inkless/backend/internal/handler/setup"
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
	"github.com/yixian-huang/inkless/backend/internal/migration"
	"github.com/yixian-huang/inkless/backend/internal/module"
	backupMod "github.com/yixian-huang/inkless/backend/internal/modules/backup"
	commentMod "github.com/yixian-huang/inkless/backend/internal/modules/comment"
	formSubmissionMod "github.com/yixian-huang/inkless/backend/internal/modules/form_submission"
	qa "github.com/yixian-huang/inkless/backend/internal/modules/qa"
	pluginruntime "github.com/yixian-huang/inkless/backend/internal/plugin"
	"github.com/yixian-huang/inkless/backend/internal/provider"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	install "github.com/yixian-huang/inkless/backend/internal/setup"
	"github.com/yixian-huang/inkless/backend/pkg/audit"
	"github.com/yixian-huang/inkless/backend/pkg/config"
	appLogger "github.com/yixian-huang/inkless/backend/pkg/logger"
	"github.com/yixian-huang/inkless/backend/pkg/secretcipher"
)

// wiredRuntime is the HTTP + background side of the process after repos/seed.
type wiredRuntime struct {
	handlers         *Handlers
	routeDeps        *RouteDeps
	setupHandler     *setupHandler.Handler
	schedulerService *service.SchedulerService
	pageViewRecorder *service.PageViewRecorder
	commentModule    *commentMod.Module
	pluginManager    *pluginruntime.Manager
	publicCache      *cache.Cache
	rbacCache        *cache.Cache
}

// wireHandlers builds services, modules, HTTP handlers, and background workers.
func wireHandlers(
	database *db.DB,
	r *repos,
	cfg *config.Config,
	build BuildInfo,
	log *appLogger.Logger,
	setupSvc *install.Service,
) (*wiredRuntime, error) {
	auditDbWriter := audit.NewDbWriter(r.auditEvent, log)
	log.Info("Audit logger initialized")

	searchService := service.NewSearchService(database.DB, db.IsPostgresDSN(cfg.DBDSN))

	registry := provider.NewRegistry()
	registry.Register("notifier", service.NewLogNotifier())
	registry.Register("captcha", &provider.NoopCaptchaProvider{})

	secretCipher, err := secretcipher.New(cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("secret cipher: %w", err)
	}

	aiConfigSvc := service.NewAIConfigService(
		repository.NewGormAIConfigRepository(database.DB),
		secretCipher,
		registry,
	)
	if err := aiConfigSvc.Restore(context.Background()); err != nil {
		return nil, fmt.Errorf("restore AI config: %w", err)
	}

	storageRuntime := service.NewStorageRuntimeService(
		r.storageConfig,
		registry,
		service.NewLocalStorage(cfg.UploadDir),
		secretCipher,
	)
	if err := storageRuntime.RestoreStartupConfig(context.Background()); err != nil {
		return nil, fmt.Errorf("restore storage config: %w", err)
	}

	pluginStore := pluginruntime.NewStore(database.DB)
	pluginManager := pluginruntime.NewManager(pluginruntime.ManagerConfig{
		PluginDir: cfg.PluginDir,
		DataDir:   cfg.PluginDataDir,
	}, pluginStore, registry)
	if cfg.ExternalPlugins {
		if err := pluginManager.StartEnabledPlugins(context.Background()); err != nil {
			return nil, fmt.Errorf("start plugins: %w", err)
		}
		pluginManager.StartHealthMonitor(30 * time.Second)
	}
	log.Info("Provider registry initialized", "providers", registry.List())

	chunkedUploadSvc := service.NewChunkedUploadServiceWithStorage(
		r.chunkedUpload,
		r.media,
		"./tmp/uploads",
		cfg.UploadDir,
		"",
		storageRuntime,
	)
	migrationSvc := migration.NewService(r.article, r.category, r.tag)

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

	publicCache := cache.New(60 * time.Second)
	rbacCache := cache.New(30 * time.Second)

	mgr := module.NewManager()
	commentModule := commentMod.New()
	mgr.Register(qa.New())
	mgr.Register(commentModule)
	mgr.Register(formSubmissionMod.New())
	mgr.Register(backupMod.New())
	if err := mgr.InitAll(module.Dependencies{
		DB:       database.DB,
		Registry: registry,
		Repos: &module.SharedRepos{
			ContentDoc: r.contentDoc,
			Article:    r.article,
		},
		SiteCfg:    r.siteConfig,
		UserRepo:   r.user,
		RBACCache:  rbacCache,
		UploadDir:  cfg.UploadDir,
		BackupDir:  cfg.BackupDir,
		AppVersion: build.Version,
	}); err != nil {
		return nil, fmt.Errorf("init modules: %w", err)
	}

	invalidateFromEvent := func(e eventbus.Event) {
		contentType, slug := "", ""
		if p, ok := e.Payload.(eventbus.ContentEventPayload); ok {
			contentType, slug = p.ContentType, p.Slug
		} else if p, ok := e.Payload.(*eventbus.ContentEventPayload); ok && p != nil {
			contentType, slug = p.ContentType, p.Slug
		}
		cache.InvalidatePublicFromContentEvent(publicCache, contentType, slug)
	}
	for _, evt := range []string{
		eventbus.ContentCreated,
		eventbus.ContentUpdated,
		eventbus.ContentDeleted,
		eventbus.ContentPublished,
		eventbus.ContentUnpublished,
		eventbus.ContentRolledBack,
	} {
		bus.Subscribe(evt, eventbus.AsyncHandler(invalidateFromEvent))
	}

	themePageService := service.NewThemePageService(r.page)
	pageViewRecorder := service.NewPageViewRecorder(r.pageView)
	pageViewRecorder.Start()

	publicHandlerInst := publicHandler.NewHandler(r.contentDoc, r.pageView, r.unifiedPage, publicCache).
		WithViewTracker(pageViewRecorder).
		WithLegacyContentDocFallback(cfg.LegacyContentDocFallback)

	unifiedPageSvc := service.NewUnifiedPageService(r.unifiedPage, r.pageVersion, bus).
		WithAuditWriter(auditDbWriter)
	articlePublicationSvc := service.NewArticlePublicationService(r.article, searchService, bus).
		WithTaxonomyRepositories(r.category, r.tag).
		WithAuditWriter(auditDbWriter)
	schedulerService := service.NewSchedulerService(r.scheduledPublishJob, articlePublicationSvc, unifiedPageSvc)
	schedulerService.Start()

	emailSvc := service.NewEmailService(r.siteConfig)
	marketplaceSvc := service.NewMarketplaceService(r.marketplace)
	wizardSvc := service.NewWizardServiceWithRegistry(registry, r.unifiedPage)
	themeExportSvc := service.NewThemeExportService(r.pageTemplate, r.siteConfig)

	apiKeySvc := service.NewAPIKeyService(database.DB)

	handlers := &Handlers{
		Auth: authHandler.NewHandler(r.user, r.refreshToken, cfg),
		Article: articleHandler.NewHandler(r.article, r.category, r.tag, searchService, bus, publicCache).
			WithPageViews(r.pageView).
			WithViewTracker(pageViewRecorder).
			WithVersionRepo(r.articleVersion),
		Public:         publicHandlerInst,
		Bootstrap:      bootstrapHandler.NewHandler(r.contentDoc, r.installedTheme, r.page, r.unifiedPage, r.siteConfig, publicCache),
		Media:          mediaHandler.NewHandlerWithStorage(r.media, cfg.UploadDir, "", storageRuntime),
		APIKey:         apiKeyHandler.NewHandler(apiKeySvc),
		Analytics:      analyticsHandler.NewHandler(r.pageView).WithCache(publicCache),
		Dashboard:      dashboardHandler.NewHandler(r.article, r.unifiedPage, r.media, r.pageView).WithCache(publicCache),
		Category:       categoryHandler.NewHandler(r.category, r.article),
		Tag:            tagHandler.NewHandler(r.tag, r.article),
		Menu:           menuHandler.NewHandler(r.menu),
		AuditLog:       auditlogHandler.NewHandler(r.auditEvent),
		Sitemap:        sitemapHandler.NewHandler(r.contentDoc, r.article, cfg.BaseURL),
		Feed:           feedHandler.NewHandler(r.article, r.siteConfig, cfg.BaseURL, "Blog", "Latest posts"),
		Theme:          themeHandler.NewHandler(r.siteConfig, publicCache),
		InstalledTheme: installedThemeHandler.NewHandler(r.installedTheme, themePageService, publicCache, r.unifiedPage),
		EmailSettings:  emailSettingsHandler.NewHandler(r.siteConfig, emailSvc),
		Features:       featuresHandler.NewHandler(r.siteConfig, publicCache),
		GlobalConfig:   globalConfigHandler.NewHandler(r.contentDoc, publicCache),
		User:           userHandler.NewHandler(r.user),
		SEO:            seoHandler.NewHandler(database.DB),
		Search:         searchhandler.NewHandler(searchService),
		Role:           roleHandler.NewHandler(r.role, r.user).WithRBACCache(rbacCache),
		Marketplace:    marketplaceHandler.NewHandler(marketplaceSvc),
		Plugin:         pluginHandler.NewHandler(pluginManager, registry, cfg.ExternalPlugins),
		Wizard:         wizardHandler.NewHandler(wizardSvc),
		AI:             aiHandler.NewHandler(registry, aiConfigSvc),
		ChunkedUpload:  chunkedUploadHandler.NewHandler(chunkedUploadSvc),
		MediaFolder:    mediaFolderHandler.NewHandler(r.mediaFolder, r.media),
		Migration:      migrationHandler.NewHandler(migrationSvc),
		Storage:        storageHandler.NewHandlerWithRuntime(storageRuntime),
		System:         systemHandler.NewHandler(database.DB, cfg.UploadDir, build.Version),
		Translation:    translationHandler.NewHandlerWithRegistry(registry, r.glossary, r.article),
		UnifiedPage:    unifiedPageHandler.NewHandler(r.unifiedPage, r.pageVersion, unifiedPageSvc, publicCache, bus),
		Scheduler:      schedulerHandler.NewHandler(schedulerService),
		PageTemplate:   pageTemplateHandler.NewHandler(r.pageTemplate),
		ThemeExport:    themeExportHandler.NewHandler(themeExportSvc),
	}

	routeDeps := &RouteDeps{
		UserRepo:       r.user,
		RBACCache:      rbacCache,
		Cfg:            cfg,
		Database:       database,
		ModuleMgr:      mgr,
		ContentDocRepo: r.contentDoc,
		AuditWriter:    auditDbWriter,
		Build:          build,
		APIKeyAuth:     apiKeySvc,
	}

	log.Info("Handlers initialized")
	return &wiredRuntime{
		handlers:         handlers,
		routeDeps:        routeDeps,
		setupHandler:     setupHandler.NewHandler(setupSvc),
		schedulerService: schedulerService,
		pageViewRecorder: pageViewRecorder,
		commentModule:    commentModule,
		pluginManager:    pluginManager,
		publicCache:      publicCache,
		rbacCache:        rbacCache,
	}, nil
}
