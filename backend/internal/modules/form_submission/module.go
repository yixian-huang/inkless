package form_submission

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/module"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
)

// Module is the self-contained form submission feature module.
type Module struct {
	handler     *Handler
	siteCfgRepo repository.SiteConfigRepository
	userRepo    repository.UserRepository
	rbacCache   *cache.Cache
}

// New creates a new form submission module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "form_submission" }

func (m *Module) Init(deps module.Dependencies) error {
	if err := deps.DB.AutoMigrate(&model.FormSubmission{}); err != nil {
		return err
	}
	repo := newGormRepository(deps.DB)
	emailSvc := service.NewEmailService(deps.SiteCfg)
	m.siteCfgRepo = deps.SiteCfg
	m.userRepo = deps.UserRepo
	m.rbacCache = deps.RBACCache
	m.handler = &Handler{
		repo:         repo,
		emailService: emailSvc,
	}
	return nil
}

// featureEnabled checks if the form_submission feature is enabled in SiteConfig.
func (m *Module) featureEnabled(c *gin.Context) bool {
	cfg, err := m.siteCfgRepo.FindByKey(c.Request.Context(), model.SiteConfigKeyFeatures)
	if err != nil || cfg == nil {
		// Default to enabled if no config exists
		return true
	}
	if cfg.PublishedConfig == nil {
		return true
	}
	fsVal, ok := cfg.PublishedConfig["form_submission"]
	if !ok {
		return true
	}
	fsMap, ok := fsVal.(map[string]interface{})
	if !ok {
		return true
	}
	enabled, ok := fsMap["enabled"]
	if !ok {
		return true
	}
	b, ok := enabled.(bool)
	return !ok || b
}

func (m *Module) RegisterRoutes(public, admin *gin.RouterGroup) {
	// Public form submission with dedicated rate limit and feature flag check
	public.POST("/form-submissions",
		middleware.FormSubmitRateLimit(),
		func(c *gin.Context) {
			if !m.featureEnabled(c) {
				c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "not found"}})
				c.Abort()
				return
			}
			c.Next()
		},
		m.handler.HandlePublicSubmit,
	)

	// Admin form submission management
	admin.GET("/form-submissions/counts", m.requirePermission("read"), m.handler.HandleAdminCounts)
	admin.GET("/form-submissions", m.requirePermission("read"), m.handler.HandleAdminList)
	admin.GET("/form-submissions/:id", m.requirePermission("read"), m.handler.HandleAdminGetByID)
	admin.PATCH("/form-submissions/:id/status", m.requirePermission("update"), m.handler.HandleAdminUpdateStatus)
	admin.POST("/form-submissions/bulk-status", m.requirePermission("update"), m.handler.HandleAdminBulkUpdateStatus)
	admin.DELETE("/form-submissions/:id", m.requirePermission("delete"), m.handler.HandleAdminDelete)
}

func (m *Module) requirePermission(action string) gin.HandlerFunc {
	return middleware.RequirePermission("form_submissions", action, m.userRepo, m.rbacCache)
}
