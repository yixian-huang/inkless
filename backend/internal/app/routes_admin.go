package app

import (
	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// requireFn builds a permission middleware for resource:action.
type requireFn func(resource, action string) gin.HandlerFunc

// registerAdminRoutes mounts all /admin domain routes after Auth + LoadRBACUser.
func registerAdminRoutes(
	admin *gin.RouterGroup,
	h *Handlers,
	require requireFn,
	userRepo repository.UserRepository,
	rbacCache *cache.Cache,
) {
	registerAdminAgent(admin, h)
	registerAdminDashboard(admin, h, require)
	registerAdminMedia(admin, h, require)
	registerAdminContent(admin, h, require, userRepo, rbacCache)
	registerAdminTaxonomy(admin, h, require)
	registerAdminMenus(admin, h, require)
	registerAdminThemes(admin, h, require)
	registerAdminSettings(admin, h, require)
	registerAdminUsersRoles(admin, h, require)
	registerAdminPlugins(admin, h, require)
	registerAdminAI(admin, h, require)
	registerAdminSystem(admin, h, require)
}

// registerAdminAgent mounts discovery endpoints for local multi-site agents.
// Auth only (no require): any valid JWT or ink_ key may probe the instance.
func registerAdminAgent(admin *gin.RouterGroup, h *Handlers) {
	if h.Agent == nil {
		return
	}
	admin.GET("/agent/whoami", h.Agent.Whoami)
}

func registerAdminDashboard(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/dashboard/summary", require("dashboard", "read"), h.Dashboard.Summary)
	analytics := admin.Group("")
	analytics.Use(require("analytics", "read"))
	analytics.GET("/analytics/summary", h.Analytics.GetSummary)

	audit := admin.Group("")
	audit.Use(require("audit_logs", "read"))
	audit.GET("/audit-logs", h.AuditLog.List)
}

func registerAdminMedia(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.POST("/media/upload", require("media", "create"), h.Media.Upload)
	admin.GET("/media", require("media", "read"), h.Media.List)
	admin.DELETE("/media/:id", require("media", "delete"), h.Media.Delete)
	admin.PUT("/media/:id/crop", require("media", "update"), h.Media.Recrop)
	admin.PUT("/media/:id", require("media", "update"), h.Media.Rename)
	admin.GET("/media/:id/usages", require("media", "read"), h.Media.GetUsages)

	admin.POST("/media/upload/init", require("media", "create"), h.ChunkedUpload.InitUpload)
	admin.POST("/media/upload/:uploadId/chunk", require("media", "create"), h.ChunkedUpload.UploadChunk)
	admin.POST("/media/upload/:uploadId/complete", require("media", "create"), h.ChunkedUpload.CompleteUpload)

	admin.GET("/media/folders", require("media", "read"), h.MediaFolder.ListTree)
	admin.POST("/media/folders", require("media", "create"), h.MediaFolder.Create)
	admin.PUT("/media/folders/:id", require("media", "update"), h.MediaFolder.Rename)
	admin.DELETE("/media/folders/:id", require("media", "delete"), h.MediaFolder.Delete)
	admin.PUT("/media/:id/move", require("media", "update"), h.MediaFolder.MoveMedia)

	// Personal API keys (PicGo / local agents). Management requires session JWT only;
	// using the key is gated by require(resource, action) ∩ key scopes.
	if h.APIKey != nil {
		keys := admin.Group("")
		keys.Use(middleware.RequireSessionJWT())
		keys.GET("/api-keys", h.APIKey.List)
		keys.POST("/api-keys", h.APIKey.Create)
		keys.DELETE("/api-keys/:id", h.APIKey.Revoke)
	}
}

func registerAdminContent(
	admin *gin.RouterGroup,
	h *Handlers,
	require requireFn,
	userRepo repository.UserRepository,
	rbacCache *cache.Cache,
) {
	admin.GET("/articles", require("articles", "read"), h.Article.AdminList)
	admin.GET("/articles/:id", require("articles", "read"), h.Article.AdminGetByID)
	admin.POST("/articles", require("articles", "create"), h.Article.AdminCreate)
	admin.PUT("/articles/:id", require("articles", "update"), h.Article.AdminUpdate)
	admin.DELETE("/articles/:id", require("articles", "delete"), h.Article.AdminDelete)
	admin.GET("/articles/:id/export", require("articles", "read"), h.Article.AdminExportMarkdown)
	admin.GET("/articles/:id/versions", require("articles", "read"), h.Article.AdminListVersions)
	admin.GET("/articles/:id/versions/compare", require("articles", "read"), h.Article.AdminCompareVersions)
	admin.GET("/articles/:id/versions/:version", require("articles", "read"), h.Article.AdminGetVersion)
	admin.POST("/articles/import", require("articles", "create"), h.Article.AdminImportMarkdown)

	sched := admin.Group("/scheduled-publications")
	sched.Use(middleware.RequireAnyPermission(
		[]middleware.PermissionPair{
			{Resource: "articles", Action: "publish"},
			{Resource: "pages", Action: "publish"},
		},
		userRepo,
		rbacCache,
	))
	{
		sched.GET("", h.Scheduler.List)
		sched.POST("", h.Scheduler.Schedule)
		sched.PUT("/:id", h.Scheduler.Reschedule)
		sched.DELETE("/:id", h.Scheduler.Cancel)
		sched.POST("/:id/retry", h.Scheduler.Retry)
	}

	admin.GET("/pages", require("pages", "read"), h.UnifiedPage.AdminList)
	admin.GET("/pages/:id", require("pages", "read"), h.UnifiedPage.AdminGetByID)
	admin.POST("/pages", require("pages", "create"), h.UnifiedPage.AdminCreate)
	admin.PUT("/pages/:id", require("pages", "update"), h.UnifiedPage.AdminUpdate)
	admin.GET("/pages/:id/draft", require("pages", "read"), h.UnifiedPage.AdminGetDraft)
	admin.PUT("/pages/:id/draft", require("pages", "update"), h.UnifiedPage.AdminUpdateDraft)
	admin.POST("/pages/:id/publish", require("pages", "publish"), h.UnifiedPage.AdminPublish)
	admin.POST("/pages/:id/unpublish", require("pages", "publish"), h.UnifiedPage.AdminUnpublish)
	admin.POST("/pages/:id/rollback", require("pages", "publish"), h.UnifiedPage.AdminRollback)
	admin.GET("/pages/:id/versions", require("pages", "read"), h.UnifiedPage.AdminListVersions)
	admin.GET("/pages/:id/versions/:version", require("pages", "read"), h.UnifiedPage.AdminGetVersionDetail)
	admin.DELETE("/pages/:id", require("pages", "delete"), h.UnifiedPage.AdminDelete)

	admin.GET("/templates", require("pages", "read"), h.PageTemplate.List)
	admin.POST("/templates", require("pages", "create"), h.PageTemplate.Create)
	admin.PUT("/templates/:id", require("pages", "update"), h.PageTemplate.Update)
	admin.DELETE("/templates/:id", require("pages", "delete"), h.PageTemplate.Delete)
	admin.POST("/templates/:id/duplicate", require("pages", "create"), h.PageTemplate.Duplicate)

	admin.POST("/wizard/generate-plan", require("pages", "create"), h.Wizard.GeneratePlan)
	admin.POST("/wizard/apply-plan", require("pages", "create"), h.Wizard.ApplyPlan)
	admin.POST("/wizard/suggest-colors", require("pages", "create"), h.Wizard.SuggestColors)
	admin.POST("/wizard/generate-content", require("pages", "create"), h.Wizard.GenerateContent)

	// Theme-bound content_documents (product-first home, corporate page keys, …).
	// Separate from unified /admin/pages (D-class). See docs/design-theme-content-admin-api.md.
	// Register static /content/slots BEFORE /content/:pageKey/* so "slots" is not a pageKey.
	if h.Content != nil {
		admin.GET("/content/slots", require("pages", "read"), h.Content.ListSlots)
		admin.GET("/content/:pageKey/schema", require("pages", "read"), h.Content.GetSchema)
		admin.GET("/content/:pageKey/draft", require("pages", "read"), h.Content.GetDraft)
		admin.PUT("/content/:pageKey/draft", require("pages", "update"), h.Content.UpdateDraft)
		admin.POST("/content/:pageKey/validate", require("pages", "update"), h.Content.Validate)
		admin.POST("/content/:pageKey/publish", require("pages", "publish"), h.Content.Publish)
		admin.POST("/content/:pageKey/rollback/:version", require("pages", "publish"), h.Content.Rollback)
		admin.GET("/content/:pageKey/versions", require("pages", "read"), h.Content.GetVersions)
		admin.GET("/content/:pageKey/versions/:version", require("pages", "read"), h.Content.GetVersionDetail)
	}
}

func registerAdminTaxonomy(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/categories", require("categories", "read"), h.Category.List)
	admin.GET("/categories/tree", require("categories", "read"), h.Category.ListTree)
	admin.GET("/categories/:id", require("categories", "read"), h.Category.GetByID)
	admin.POST("/categories", require("categories", "create"), h.Category.Create)
	admin.PUT("/categories/:id", require("categories", "update"), h.Category.Update)
	admin.DELETE("/categories/:id", require("categories", "delete"), h.Category.Delete)

	admin.GET("/tags", require("tags", "read"), h.Tag.List)
	admin.POST("/tags", require("tags", "create"), h.Tag.Create)
	admin.PUT("/tags/:id", require("tags", "update"), h.Tag.Update)
	admin.DELETE("/tags/:id", require("tags", "delete"), h.Tag.Delete)
}

func registerAdminMenus(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/menus", require("menus", "read"), h.Menu.ListGroups)
	admin.POST("/menus", require("menus", "create"), h.Menu.CreateGroup)
	admin.GET("/menus/:id", require("menus", "read"), h.Menu.GetGroup)
	admin.PUT("/menus/:id", require("menus", "update"), h.Menu.UpdateGroup)
	admin.DELETE("/menus/:id", require("menus", "delete"), h.Menu.DeleteGroup)
	admin.PUT("/menus/:id/primary", require("menus", "update"), h.Menu.SetPrimary)
	admin.POST("/menus/:id/items", require("menus", "update"), h.Menu.CreateItem)
	admin.PUT("/menus/:id/items/:itemId", require("menus", "update"), h.Menu.UpdateItem)
	admin.DELETE("/menus/:id/items/:itemId", require("menus", "update"), h.Menu.DeleteItem)
	admin.PUT("/menus/:id/items/reorder", require("menus", "update"), h.Menu.ReorderItems)
}

func registerAdminThemes(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/theme", require("themes", "read"), h.Theme.AdminGet)
	admin.PUT("/theme", require("themes", "update"), h.Theme.AdminUpdate)

	admin.GET("/themes", require("themes", "read"), h.InstalledTheme.AdminList)
	admin.GET("/themes/:id", require("themes", "read"), h.InstalledTheme.AdminGetByID)
	admin.POST("/themes", require("themes", "create"), h.InstalledTheme.AdminCreate)
	admin.PUT("/themes/:id", require("themes", "update"), h.InstalledTheme.AdminUpdate)
	admin.DELETE("/themes/:id", require("themes", "delete"), h.InstalledTheme.AdminDelete)
	admin.PUT("/themes/:id/activate", require("themes", "manage"), h.InstalledTheme.AdminActivate)

	admin.POST("/theme-packages/export", require("themes", "manage"), h.ThemeExport.Export)
	admin.POST("/theme-packages/import", require("themes", "manage"), h.ThemeExport.Import)
	admin.GET("/theme-packages", require("themes", "manage"), h.ThemeExport.List)
	admin.PUT("/theme-packages/:id/apply", require("themes", "manage"), h.ThemeExport.Apply)

	// Official theme catalog / install / auto-update (Phase A extension store)
	if h.Extensions != nil {
		admin.GET("/extensions/themes/catalog", require("themes", "read"), h.Extensions.AdminThemeCatalog)
		admin.POST("/extensions/themes/install", require("themes", "manage"), h.Extensions.AdminThemeInstall)
		admin.GET("/extensions/themes/auto-update", require("themes", "read"), h.Extensions.AdminThemeAutoUpdateGet)
		admin.PUT("/extensions/themes/auto-update", require("themes", "manage"), h.Extensions.AdminThemeAutoUpdatePut)
		admin.POST("/extensions/themes/auto-update/run", require("themes", "manage"), h.Extensions.AdminThemeAutoUpdateRun)
	}
}

func registerAdminSettings(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/email-settings", require("settings", "read"), h.EmailSettings.HandleGet)
	admin.PUT("/email-settings", require("settings", "manage"), h.EmailSettings.HandleUpdate)
	admin.POST("/email-settings/test", require("settings", "manage"), h.EmailSettings.HandleTest)

	globalConfigAdmin := admin.Group("")
	globalConfigAdmin.Use(require("settings", "manage"))
	h.GlobalConfig.RegisterRoutes(globalConfigAdmin)

	featuresAdmin := admin.Group("")
	featuresAdmin.Use(require("settings", "manage"))
	h.Features.RegisterRoutes(featuresAdmin)

	admin.GET("/storage/config", require("settings", "manage"), h.Storage.GetConfig)
	admin.PUT("/storage/config", require("settings", "manage"), h.Storage.UpdateConfig)
	admin.POST("/storage/test", require("settings", "manage"), h.Storage.TestConnection)

	// Article authors need translate in the editor; settings:manage was too narrow.
	admin.POST("/translate", require("articles", "update"), h.Translation.Translate)
	admin.POST("/translate/batch", require("articles", "update"), h.Translation.BatchTranslate)
	admin.POST("/translate/article/:id", require("articles", "update"), h.Translation.TranslateArticle)
	admin.GET("/glossary", require("settings", "manage"), h.Translation.GlossaryList)
	admin.POST("/glossary", require("settings", "manage"), h.Translation.GlossaryCreate)
	admin.PUT("/glossary/:id", require("settings", "manage"), h.Translation.GlossaryUpdate)
	admin.DELETE("/glossary/:id", require("settings", "manage"), h.Translation.GlossaryDelete)
}

func registerAdminUsersRoles(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	users := admin.Group("/users")
	{
		users.GET("", require("users", "read"), h.User.List)
		users.GET("/:id", require("users", "read"), h.User.GetByID)
		users.POST("", require("users", "create"), h.User.Create)
		users.PUT("/:id", require("users", "update"), h.User.Update)
		users.DELETE("/:id", require("users", "delete"), h.User.Delete)
	}

	roles := admin.Group("/roles")
	{
		roles.GET("", require("roles", "read"), h.Role.List)
		roles.GET("/:id", require("roles", "read"), h.Role.GetByID)
		roles.POST("", require("roles", "create"), h.Role.Create)
		roles.PUT("/:id", require("roles", "update"), h.Role.Update)
		roles.DELETE("/:id", require("roles", "delete"), h.Role.Delete)
		roles.POST("/assign", require("roles", "manage"), h.Role.AssignRole)
		roles.POST("/unassign", require("roles", "manage"), h.Role.UnassignRole)
		roles.GET("/user/:userId", require("roles", "read"), h.Role.GetUserRoles)
	}
	admin.GET("/permissions", require("roles", "read"), h.Role.ListPermissions)
}

func registerAdminPlugins(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/marketplace/items", require("plugins", "read"), h.Marketplace.AdminListItems)
	admin.GET("/marketplace/installed", require("plugins", "read"), h.Marketplace.AdminListInstalled)
	admin.POST("/marketplace/items", require("plugins", "manage"), h.Marketplace.AdminRegisterItem)
	admin.GET("/marketplace/items/:slug", require("plugins", "read"), h.Marketplace.AdminGetItem)
	admin.POST("/marketplace/items/:slug/install", require("plugins", "manage"), h.Marketplace.AdminInstallItem)
	admin.PUT("/marketplace/items/:slug/update", require("plugins", "manage"), h.Marketplace.AdminUpdateItem)
	admin.DELETE("/marketplace/items/:slug", require("plugins", "manage"), h.Marketplace.AdminUninstallItem)
	admin.POST("/marketplace/items/:slug/versions", require("plugins", "manage"), h.Marketplace.AdminAddVersion)

	admin.GET("/plugins", require("plugins", "read"), h.Plugin.List)
	admin.POST("/plugins/install", require("system", "manage"), h.Plugin.Install)
	admin.POST("/plugins/:id/enable", require("system", "manage"), h.Plugin.Enable)
	admin.POST("/plugins/:id/disable", require("system", "manage"), h.Plugin.Disable)
	admin.DELETE("/plugins/:id", require("system", "manage"), h.Plugin.Uninstall)
	admin.PUT("/plugins/:id/settings", require("system", "manage"), h.Plugin.UpdateSettings)
	admin.POST("/plugins/test-notification", require("system", "manage"), h.Plugin.TestNotification)
}

func registerAdminAI(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.POST("/ai/chat", require("settings", "manage"), h.AI.Chat)
	admin.POST("/ai/summarize", require("settings", "manage"), h.AI.Summarize)
	admin.POST("/ai/suggest-titles", require("settings", "manage"), h.AI.SuggestTitles)
	admin.POST("/ai/suggest-tags", require("settings", "manage"), h.AI.SuggestTags)
	admin.POST("/ai/complete", require("settings", "manage"), h.AI.Complete)
	// Article editor metadata: authors with update permission (not settings:manage).
	admin.POST("/ai/article-meta", require("articles", "update"), h.AI.ArticleMeta)
	admin.GET("/ai/config", require("settings", "manage"), h.AI.GetConfig)
	admin.PUT("/ai/config", require("settings", "manage"), h.AI.UpdateConfig)
	admin.POST("/ai/config/test", require("settings", "manage"), h.AI.TestConfig)
}

func registerAdminSystem(admin *gin.RouterGroup, h *Handlers, require requireFn) {
	admin.GET("/system/status", require("system", "manage"), h.System.GetStatus)

	// Host self-update (H0 probe + H1 apply). See docs/design-host-self-update-mvp.md
	admin.GET("/system/update", require("system", "manage"), h.System.GetUpdate)
	admin.POST("/system/update/check", require("system", "manage"), h.System.CheckUpdate)
	admin.POST("/system/update/apply", require("system", "manage"), h.System.ApplyUpdate)
	admin.GET("/system/update/jobs/:id", require("system", "manage"), h.System.GetUpdateJob)
	admin.POST("/system/update/rollback", require("system", "manage"), h.System.RollbackUpdate)

	admin.POST("/migration/import", require("system", "manage"), h.Migration.Import)
	admin.GET("/migration/jobs", require("system", "manage"), h.Migration.ListJobs)
	admin.GET("/migration/jobs/:jobId", require("system", "manage"), h.Migration.GetJob)
	admin.POST("/migration/jobs/:jobId/retry", require("system", "manage"), h.Migration.RetryJob)
	admin.GET("/migration/jobs/:jobId/stream", require("system", "manage"), h.Migration.StreamProgress)
}
