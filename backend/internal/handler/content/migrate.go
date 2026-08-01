package content

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// MigrateToPagesRequest is POST /admin/content/migrate-to-pages body.
type MigrateToPagesRequest struct {
	// Force overwrites Page draft+published from content_documents.
	Force bool `json:"force"`
	// ThemeID optional; defaults to active theme.
	ThemeID string `json:"themeId"`
}

// MigrateToPages runs content_documents → unified_pages migration (T3).
// @Summary      Migrate content_documents to unified pages
// @Description  Ensures home Page exists; optional force-sync from content_documents. Deprecated content API remains as bridge.
// @Tags         Content (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body MigrateToPagesRequest false "Options"
// @Success      200 {object} object{results=[]service.ContentToPageResult}
// @Router       /admin/content/migrate-to-pages [post]
func (h *Handler) MigrateToPages(c *gin.Context) {
	if h.pageRepo == nil {
		apierror.Write(c, apierror.InternalServerError("page bridge not configured"))
		return
	}
	var req MigrateToPagesRequest
	_ = c.ShouldBindJSON(&req)

	// Need theme repo via migrator — construct from slots resolver's theme path if available.
	// Handler only has pageRepo; theme id from request or empty → migrator finds active.
	migrator := service.NewContentToPageMigrator(h.pageRepo, h.docRepo, nil)
	// When themeID empty and theme repo nil, MigrateHome still works if themeID passed.
	if req.ThemeID == "" && h.slots != nil {
		res := h.slots.ResolveActive(c.Request.Context())
		req.ThemeID = res.ActiveThemeID
	}
	results := migrator.MigrateAll(c.Request.Context(), req.ThemeID, req.Force)
	service.LogMigration(results)
	setContentDeprecationHeaders(c, "home")
	c.JSON(http.StatusOK, gin.H{"results": results})
}
