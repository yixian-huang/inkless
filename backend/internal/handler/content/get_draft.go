package content

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// GetDraftResponse is GET /admin/content/{pageKey}/draft.
type GetDraftResponse struct {
	PageKey          string        `json:"pageKey"`
	Version          int           `json:"version"`
	Config           model.JSONMap `json:"config"`
	PublishedVersion int           `json:"publishedVersion"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	// Storage indicates where the draft was loaded from (page | content_documents).
	Storage string `json:"storage,omitempty"`
	PageID  *uint  `json:"pageId,omitempty"`
}

// GetDraft returns the draft config (empty config + version 0 if missing).
// Prefers unified Page when present (theme-as-templates T3).
// @Summary      Get theme content draft
// @Description  Deprecated: prefer /admin/pages. Returns draft from Page when bridged, else content_documents.
// @Tags         Content (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey path string true "Page key (home, contact, …)"
// @Success      200 {object} GetDraftResponse
// @Failure      400 {object} object{error=object}
// @Router       /admin/content/{pageKey}/draft [get]
func (h *Handler) GetDraft(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}
	setContentDeprecationHeaders(c, string(pageKey))

	// T3 bridge: prefer unified page
	if page := h.findBridgePage(c.Request.Context(), pageKey); page != nil {
		cfg := model.JSONMap(page.DraftConfig)
		if cfg == nil {
			cfg = model.JSONMap{}
		}
		id := page.ID
		c.JSON(http.StatusOK, GetDraftResponse{
			PageKey:          string(pageKey),
			Version:          page.DraftVersion,
			Config:           cfg,
			PublishedVersion: page.PublishedVersion,
			UpdatedAt:        page.UpdatedAt,
			Storage:          "page",
			PageID:           &id,
		})
		return
	}

	doc, err := h.docRepo.FindByPageKey(c.Request.Context(), pageKey)
	if err != nil {
		if isNotFoundErr(err) {
			c.JSON(http.StatusOK, emptyDraftResponse(pageKey))
			return
		}
		apierror.Write(c, apierror.InternalServerError("Failed to fetch draft"))
		return
	}

	cfg := doc.DraftConfig
	if cfg == nil {
		cfg = model.JSONMap{}
	}
	c.JSON(http.StatusOK, GetDraftResponse{
		PageKey:          string(doc.PageKey),
		Version:          doc.DraftVersion,
		Config:           cfg,
		PublishedVersion: doc.PublishedVersion,
		UpdatedAt:        doc.UpdatedAt,
		Storage:          "content_documents",
	})
}
