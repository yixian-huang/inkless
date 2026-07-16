package qa

import (
	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/module"
	"blotting-consultancy/internal/repository"
)

type Module struct {
	handler   *Handler
	userRepo  repository.UserRepository
	rbacCache *cache.Cache
}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "qa" }

func (m *Module) Init(deps module.Dependencies) error {
	if err := deps.DB.AutoMigrate(&QALog{}); err != nil {
		return err
	}
	qaLogRepo := newGormQALogRepository(deps.DB)
	vectorStore := NewMemoryVectorStore()
	qaService := NewQAService(deps.Registry.AI(), vectorStore)
	embeddingService := NewEmbeddingService(deps.Registry.AI(), vectorStore)
	m.handler = &Handler{
		qaService:        qaService,
		embeddingService: embeddingService,
		qaLogRepo:        qaLogRepo,
		contentDocRepo:   deps.Repos.ContentDoc,
		articleRepo:      deps.Repos.Article,
		siteCfgRepo:      deps.SiteCfg,
	}
	m.userRepo = deps.UserRepo
	m.rbacCache = deps.RBACCache
	return nil
}

func (m *Module) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.POST("/qa/ask", m.handler.PublicAsk)
	admin.POST("/qa/index", m.requirePermission("manage"), m.handler.AdminIndex)
	admin.GET("/qa/logs", m.requirePermission("read"), m.handler.AdminListLogs)
	admin.POST("/qa/logs/:id/feedback", m.requirePermission("manage"), m.handler.AdminFeedback)
}

func (m *Module) requirePermission(action string) gin.HandlerFunc {
	return middleware.RequirePermission("settings", action, m.userRepo, m.rbacCache)
}
