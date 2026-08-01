package content

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// UpdateDraftRequest is PUT /admin/content/{pageKey}/draft body.
type UpdateDraftRequest struct {
	Config     model.JSONMap `json:"config" binding:"required"`
	ChangeNote string        `json:"changeNote"`
}

// UpdateDraftResponse is the PUT draft response.
type UpdateDraftResponse struct {
	PageKey   string    `json:"pageKey"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UpdateDraft updates draft with If-Match optimistic locking.
// MediaRef string-leaf violations return 400.
// Missing document + If-Match: 0 creates a new document at version 1.
// @Summary      Update theme content draft
// @Description  Optimistic lock via If-Match draft version; MediaRef url/alt/caption must be strings
// @Tags         Content (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey  path   string true "Page key"
// @Param        If-Match header string true "Expected draft version"
// @Param        body     body   UpdateDraftRequest true "Draft body"
// @Success      200 {object} UpdateDraftResponse
// @Failure      400 {object} object{error=object}
// @Failure      409 {object} object{error=object}
// @Router       /admin/content/{pageKey}/draft [put]
func (h *Handler) UpdateDraft(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	ifMatchHeader := c.GetHeader("If-Match")
	if ifMatchHeader == "" {
		apierror.Write(c, apierror.BadRequest("If-Match header is required"))
		return
	}
	expectedVersion, err := strconv.Atoi(ifMatchHeader)
	if err != nil {
		apierror.Write(c, apierror.BadRequest("If-Match header must be a valid integer"))
		return
	}

	var req UpdateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body"))
		return
	}
	if req.Config == nil {
		req.Config = model.JSONMap{}
	}

	setContentDeprecationHeaders(c, string(pageKey))

	// Hard reject invalid MediaRef leaves on write (prevents React #31 in themes).
	if mediaErrs := service.CollectMediaRefLeafErrors(req.Config); len(mediaErrs) > 0 {
		details := make([]map[string]string, 0, len(mediaErrs))
		for _, e := range mediaErrs {
			details = append(details, map[string]string{
				"path":    e.Path,
				"code":    e.Code,
				"message": e.Message,
			})
		}
		apierror.Write(c, apierror.New(http.StatusBadRequest, "MEDIAREF_TYPE", "MediaRef url/alt/caption must be strings").
			WithDetails(map[string]any{"errors": details}))
		return
	}

	ctx := c.Request.Context()

	// T3 bridge: write unified Page draft when present; dual-write content_documents.
	if page := h.findBridgePage(ctx, pageKey); page != nil {
		newVersion, err := h.pageRepo.UpdateDraft(ctx, page.ID, expectedVersion, req.Config)
		if err != nil {
			if errors.Is(err, repository.ErrUnifiedPageDraftVersionConflict) {
				apierror.Write(c, apierror.New(http.StatusConflict, "CONFLICT_VERSION", "Draft version conflict"))
				return
			}
			apierror.Write(c, apierror.InternalServerError("Failed to update page draft: "+err.Error()))
			return
		}
		// dual-write legacy content_documents for public dual-read
		if fresh, fe := h.pageRepo.FindByID(ctx, page.ID); fe == nil && fresh != nil {
			_ = service.SyncContentDocumentFromPage(ctx, h.docRepo, fresh, pageKey)
		}
		invalidateContentCache(h.publicCache, string(pageKey))
		c.JSON(http.StatusOK, UpdateDraftResponse{
			PageKey:   string(pageKey),
			Version:   newVersion,
			UpdatedAt: time.Now().UTC(),
		})
		return
	}

	_, findErr := h.docRepo.FindByPageKey(ctx, pageKey)
	if findErr != nil && isNotFoundErr(findErr) {
		if expectedVersion != 0 {
			apierror.Write(c, apierror.NotFound("Content document not found"))
			return
		}
		doc := &model.ContentDocument{
			PageKey:          pageKey,
			DraftConfig:      req.Config,
			DraftVersion:     1,
			PublishedConfig:  model.JSONMap{},
			PublishedVersion: 0,
		}
		if err := h.docRepo.Create(ctx, doc); err != nil {
			apierror.Write(c, apierror.InternalServerError("Failed to create draft document"))
			return
		}
		c.JSON(http.StatusOK, UpdateDraftResponse{
			PageKey:   string(pageKey),
			Version:   1,
			UpdatedAt: time.Now().UTC(),
		})
		return
	}
	if findErr != nil {
		apierror.Write(c, apierror.InternalServerError("Failed to load draft"))
		return
	}

	newVersion, err := h.docRepo.UpdateDraft(ctx, pageKey, expectedVersion, req.Config)
	if err != nil {
		if err.Error() == "draft version conflict or document not found" {
			apierror.Write(c, apierror.New(http.StatusConflict, "CONFLICT_VERSION", "Draft version conflict"))
			return
		}
		if isNotFoundErr(err) {
			apierror.Write(c, apierror.NotFound("Content document not found"))
			return
		}
		apierror.Write(c, apierror.InternalServerError("Failed to update draft"))
		return
	}

	c.JSON(http.StatusOK, UpdateDraftResponse{
		PageKey:   string(pageKey),
		Version:   newVersion,
		UpdatedAt: time.Now().UTC(),
	})
}
