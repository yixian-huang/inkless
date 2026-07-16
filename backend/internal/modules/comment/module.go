package comment

import (
	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/module"
	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/repository"
)

// Module is the self-contained comment feature module.
type Module struct {
	handler   *Handler
	antispam  *AntiSpamService
	userRepo  repository.UserRepository
	rbacCache *cache.Cache
}

// New creates a new comment module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "comment" }

func (m *Module) Init(deps module.Dependencies) error {
	if err := deps.DB.AutoMigrate(&Comment{}); err != nil {
		return err
	}
	repo := newGormRepository(deps.DB)
	captcha := &provider.NoopCaptchaProvider{}
	m.antispam = newAntiSpamService(captcha)
	var contentDoc repository.ContentDocumentRepository
	if deps.Repos != nil {
		contentDoc = deps.Repos.ContentDoc
	}
	m.handler = &Handler{
		repo:           repo,
		antispam:       m.antispam,
		siteCfgRepo:    deps.SiteCfg,
		contentDocRepo: contentDoc,
	}
	m.userRepo = deps.UserRepo
	m.rbacCache = deps.RBACCache
	return nil
}

func (m *Module) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.POST("/comments", m.handler.PublicCreate)
	public.GET("/comments", m.handler.PublicList)
	admin.GET("/comments", m.requirePermission("read"), m.handler.AdminList)
	admin.PATCH("/comments/:id/status", m.requirePermission("update"), m.handler.AdminUpdateStatus)
	admin.DELETE("/comments/:id", m.requirePermission("delete"), m.handler.AdminDelete)
	admin.PUT("/comments/:id/pin", m.requirePermission("update"), m.handler.AdminPin)
	admin.POST("/comments/reply", m.requirePermission("create"), m.handler.AdminReply)
}

func (m *Module) requirePermission(action string) gin.HandlerFunc {
	return middleware.RequirePermission("comments", action, m.userRepo, m.rbacCache)
}

// Stop shuts down background goroutines (antispam cleanup).
func (m *Module) Stop() {
	if m.antispam != nil {
		m.antispam.Stop()
	}
}
