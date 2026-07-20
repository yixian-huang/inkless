package wizard

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// Handler handles AI site building wizard HTTP requests.
type Handler struct {
	svc *service.WizardService
}

// NewHandler creates a new wizard Handler.
func NewHandler(svc *service.WizardService) *Handler {
	return &Handler{svc: svc}
}

// GeneratePlan handles POST /admin/wizard/generate-plan
//
// Accepts a questionnaire (industry, style, features, etc.) and returns
// an AI-generated site plan with recommended theme, pages, and color scheme.
func (h *Handler) GeneratePlan(c *gin.Context) {
	var q model.Questionnaire
	if err := c.ShouldBindJSON(&q); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	plan, err := h.svc.GenerateSitePlan(c.Request.Context(), q)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, plan)
}

// ApplyPlan handles POST /admin/wizard/apply-plan
//
// Accepts a SitePlan and creates pages in the CMS. Existing pages (by slug)
// are skipped. Returns a summary of created and skipped pages.
func (h *Handler) ApplyPlan(c *gin.Context) {
	var plan model.SitePlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	result, err := h.svc.ScaffoldSite(c.Request.Context(), plan)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// SuggestColors handles POST /admin/wizard/suggest-colors
//
// Accepts an industry name and optional brand name, then returns an AI-recommended
// color palette suited to that brand identity.
func (h *Handler) SuggestColors(c *gin.Context) {
	var req model.ColorSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	scheme, err := h.svc.SuggestColors(c.Request.Context(), req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, scheme)
}

// GenerateContent handles POST /admin/wizard/generate-content
//
// Accepts a page type (e.g., "home", "about", "services") and industry, then
// returns AI-generated sample copy suitable for that page.
func (h *Handler) GenerateContent(c *gin.Context) {
	var req model.GenerateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	content, err := h.svc.GenerateContent(c.Request.Context(), req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, content)
}

// handleServiceError maps service-level errors to appropriate HTTP responses.
func (h *Handler) handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrAINotConfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "AI_NOT_CONFIGURED",
				"message": "AI provider is not configured. Please configure an AI provider in settings.",
			},
		})
		return
	}

	c.JSON(http.StatusBadGateway, gin.H{
		"error": gin.H{
			"code":    "WIZARD_ERROR",
			"message": err.Error(),
		},
	})
}
