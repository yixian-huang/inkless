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
}

// GetDraft returns the draft config (empty config + version 0 if missing).
func (h *Handler) GetDraft(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
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
	})
}
