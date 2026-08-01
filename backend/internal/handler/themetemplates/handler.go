package themetemplates

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/themetemplates"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// Handler serves theme template discovery (T4).
type Handler struct {
	resolver *themetemplates.Resolver
}

// NewHandler creates a templates discovery handler.
func NewHandler(resolver *themetemplates.Resolver) *Handler {
	return &Handler{resolver: resolver}
}

// ListActive returns templates for the active theme.
// @Summary      List active theme templates
// @Description  templates[] or contentSlots projection for theme-as-templates agents
// @Tags         Themes
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object
// @Router       /admin/themes/active/templates [get]
func (h *Handler) ListActive(c *gin.Context) {
	if h == nil || h.resolver == nil {
		c.JSON(http.StatusOK, gin.H{
			"source": "none", "templates": []any{}, "defaultTemplates": gin.H{},
		})
		return
	}
	res := h.resolver.ResolveActive(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"activeThemeId":      res.ActiveThemeID,
		"activeThemeVersion": res.ActiveThemeVersion,
		"source":             res.Source,
		"templates":          themetemplates.Summarize(res.Templates),
		"defaultTemplates":   res.DefaultTemplates,
	})
}

// GetActive returns one template by key (query ?key=product-first/home@1).
// @Summary      Get active theme template by key
// @Tags         Themes
// @Produce      json
// @Security     BearerAuth
// @Param        key query string true "Template key"
// @Success      200 {object} themetemplates.Template
// @Failure      404 {object} object{error=object}
// @Router       /admin/themes/active/template [get]
func (h *Handler) GetActive(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		apierror.Write(c, apierror.BadRequest("query key is required"))
		return
	}
	if h == nil || h.resolver == nil {
		apierror.Write(c, apierror.NotFound("template not found"))
		return
	}
	res, tmpl, ok := h.resolver.ResolveTemplate(c.Request.Context(), key)
	if !ok {
		apierror.Write(c, apierror.NotFound("template not found for active theme"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"activeThemeId":      res.ActiveThemeID,
		"activeThemeVersion": res.ActiveThemeVersion,
		"source":             res.Source,
		"template":           tmpl,
	})
}
