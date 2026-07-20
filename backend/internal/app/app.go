package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/db"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	commentMod "github.com/yixian-huang/inkless/backend/internal/modules/comment"
	pluginruntime "github.com/yixian-huang/inkless/backend/internal/plugin"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	"github.com/yixian-huang/inkless/backend/pkg/brand"
	"github.com/yixian-huang/inkless/backend/pkg/config"
	appLogger "github.com/yixian-huang/inkless/backend/pkg/logger"
)

// App is the fully wired Inkless CMS process (DB, HTTP, background workers).
type App struct {
	Build BuildInfo
	Cfg   *config.Config
	Log   *appLogger.Logger

	database         *db.DB
	router           *gin.Engine
	schedulerService *service.SchedulerService
	pageViewRecorder *service.PageViewRecorder
	commentModule    *commentMod.Module
	pluginManager    *pluginruntime.Manager
	publicCache      *cache.Cache
	rbacCache        *cache.Cache
}

// Options configures application bootstrap.
type Options struct {
	Build BuildInfo
}

// New loads infrastructure, runs migrations/seed, wires handlers, and builds the HTTP router.
// It does not start listening; call Run.
//
// Bootstrap order (fixed): openDB → migrate → repos → seed → handlers → router.
func New(loadResult *config.LoadResult, opts Options) (*App, error) {
	if loadResult == nil || loadResult.Config == nil {
		return nil, fmt.Errorf("app: nil config load result")
	}
	build := opts.Build
	if build.Version == "" {
		build = DefaultBuildInfo()
	}
	cfg := loadResult.Config

	log := appLogger.New(cfg.Env, map[string]interface{}{
		"service": brand.APIService,
		"version": build.Version,
	})
	log.Info("Starting server",
		"env", cfg.Env,
		"port", cfg.Port,
		"version", build.Version,
		"buildTime", build.BuildTime,
		"gitCommit", build.GitCommit,
		"gitBranch", build.GitBranch,
		"bootstrapMode", loadResult.BootstrapMode,
		"legacyContentDocFallback", cfg.LegacyContentDocFallback,
	)
	if loadResult.BootstrapMode {
		log.Warn("Setup bootstrap mode active — use /setup to persist .env and restart")
	}

	maxOpenConn := 25
	maxIdleConn := 5
	maxLifetime := 5 * time.Minute
	if !db.IsPostgresDSN(cfg.DBDSN) {
		maxOpenConn = 4
		maxIdleConn = 2
		maxLifetime = 0
	}

	database, err := db.Init(db.InitOptions{
		DSN:         cfg.DBDSN,
		MaxOpenConn: maxOpenConn,
		MaxIdleConn: maxIdleConn,
		MaxLifetime: maxLifetime,
		LogLevel:    gormLogLevel(cfg.Env),
	})
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	log.Info("Database connection established")

	if err := migrateSchema(database, log); err != nil {
		_ = database.Close()
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.HealthCheck(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("database health check: %w", err)
	}
	log.Info("Database health check passed")

	r := wireRepos(database.DB)
	log.Info("Repositories initialized")

	setupSvc, err := runSeedAndSetup(database, r, cfg, loadResult, log)
	if err != nil {
		_ = database.Close()
		return nil, err
	}

	rt, err := wireHandlers(database, r, cfg, build, log, setupSvc)
	if err != nil {
		_ = database.Close()
		return nil, err
	}

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(log, middleware.RequestLoggerOptions{}))
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{
		"/health",
		"/healthz",
		"/ready",
		"/metrics",
	}), gzip.WithExcludedExtensions([]string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".woff", ".woff2", ".zip", ".gz",
	})))
	router.Use(middleware.AuditContext())

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
	router.Use(apierror.ErrorHandler())

	registerRoutes(router, rt.handlers, rt.routeDeps)
	rt.setupHandler.RegisterRoutes(router, middleware.LoginRateLimit())
	log.Info("Router configured with all routes")

	return &App{
		Build:            build,
		Cfg:              cfg,
		Log:              log,
		database:         database,
		router:           router,
		schedulerService: rt.schedulerService,
		pageViewRecorder: rt.pageViewRecorder,
		commentModule:    rt.commentModule,
		pluginManager:    rt.pluginManager,
		publicCache:      rt.publicCache,
		rbacCache:        rt.rbacCache,
	}, nil
}

// Handler returns the HTTP handler for tests or custom servers.
func (a *App) Handler() http.Handler {
	return a.router
}

// Run starts the HTTP server and blocks until SIGINT/SIGTERM, then shuts down cleanly.
func (a *App) Run() error {
	addr := fmt.Sprintf(":%d", a.Cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		a.Log.Info("Server listening", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		a.Log.Info("Server shutting down gracefully...", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			_ = a.shutdownWorkers()
			return fmt.Errorf("server listen: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		a.Log.Error("Server forced to shutdown", "error", err)
	}
	return a.shutdownWorkers()
}

func (a *App) shutdownWorkers() error {
	if a.schedulerService != nil {
		a.schedulerService.Stop()
	}
	if a.pageViewRecorder != nil {
		a.pageViewRecorder.Stop(3 * time.Second)
	}
	if a.commentModule != nil {
		a.commentModule.Stop()
	}
	if a.pluginManager != nil {
		if err := a.pluginManager.StopAll(); err != nil {
			a.Log.Error("Failed to stop plugins cleanly", "error", err)
		}
	}
	if a.publicCache != nil {
		a.publicCache.Stop()
	}
	if a.rbacCache != nil {
		a.rbacCache.Stop()
	}
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			a.Log.Error("Failed to close database connection", "error", err)
			return err
		}
	}
	a.Log.Info("Server stopped")
	return nil
}
