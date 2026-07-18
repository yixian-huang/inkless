package backup

import (
	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/module"
	"blotting-consultancy/internal/repository"
)

// Module is the self-contained backup feature module.
type Module struct {
	handler   *Handler
	service   *Service
	userRepo  repository.UserRepository
	rbacCache *cache.Cache
}

// New creates a new backup module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "backup" }

func (m *Module) Init(deps module.Dependencies) error {
	m.service = NewService(deps.DB, deps.BackupDir, 10, deps.UploadDir, deps.AppVersion)
	m.handler = &Handler{service: m.service}
	m.userRepo = deps.UserRepo
	m.rbacCache = deps.RBACCache
	return nil
}

func (m *Module) RegisterRoutes(_ *gin.RouterGroup, admin *gin.RouterGroup) {
	// Basic backup management
	admin.GET("/backups", m.requirePermission("read"), m.handler.List)
	admin.POST("/backups/trigger", m.requirePermission("create"), m.handler.Trigger)

	// Site export/import (requires backups:manage via RBAC)
	backupGroup := admin.Group("/backups")
	backupGroup.Use(m.requirePermission("manage"))
	{
		backupGroup.POST("/export", m.handler.Export)
		backupGroup.GET("/export/:filename", m.handler.DownloadExport)
		backupGroup.POST("/import", m.handler.Import)
		backupGroup.POST("/import/validate", m.handler.ValidateImport)
	}
}

func (m *Module) requirePermission(action string) gin.HandlerFunc {
	return middleware.RequirePermission("backups", action, m.userRepo, m.rbacCache)
}

// Service returns the underlying backup service (for use in graceful shutdown, etc.)
func (m *Module) Service() *Service {
	return m.service
}
