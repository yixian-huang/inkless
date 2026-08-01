package content

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	"github.com/yixian-huang/inkless/backend/pkg/metrics"
)

// PublishRequest is POST /admin/content/{pageKey}/publish body.
type PublishRequest struct {
	ExpectedDraftVersion int    `json:"expectedDraftVersion" binding:"required"`
	ChangeNote           string `json:"changeNote"`
}

// PublishResponse is the publish success payload.
type PublishResponse struct {
	PageKey          string    `json:"pageKey"`
	PublishedVersion int       `json:"publishedVersion"`
	PublishedAt      time.Time `json:"publishedAt"`
}

// Publish promotes draft to published and invalidates public content cache.
// @Summary      Publish theme content
// @Description  Promotes draft to published with expectedDraftVersion lock; invalidates content cache
// @Tags         Content (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey path string true "Page key"
// @Param        body    body PublishRequest true "Publish parameters"
// @Success      200 {object} PublishResponse
// @Failure      409 {object} object{error=object}
// @Failure      422 {object} object{error=object}
// @Router       /admin/content/{pageKey}/publish [post]
func (h *Handler) Publish(c *gin.Context) {
	pageKeyStr := c.Param("pageKey")
	pageKey := model.PageKey(pageKeyStr)
	metrics.Global().RecordPublishAttempt()

	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body"))
		return
	}

	user := middleware.GetUserContext(c)
	if user == nil {
		apierror.Write(c, apierror.Unauthorized("User context not found"))
		return
	}
	setContentDeprecationHeaders(c, pageKeyStr)

	// T3 bridge: publish unified Page when present
	if page := h.findBridgePage(c.Request.Context(), pageKey); page != nil && h.pageSvc != nil {
		err := h.pageSvc.Publish(c.Request.Context(), page.ID, req.ExpectedDraftVersion, user.UserID)
		if err != nil {
			metrics.Global().RecordPublishFailure()
			h.logPublishFail(pageKeyStr, user.Username, err, req.ExpectedDraftVersion)
			if errors.Is(err, service.ErrPageVersionConflict) {
				apierror.Write(c, apierror.New(http.StatusConflict, "CONFLICT_VERSION", "Draft version mismatch"))
				return
			}
			apierror.Write(c, apierror.InternalServerError("Failed to publish page: "+err.Error()))
			return
		}
		fresh, _ := h.pageRepo.FindByID(c.Request.Context(), page.ID)
		if fresh != nil {
			_ = service.SyncContentDocumentFromPage(c.Request.Context(), h.docRepo, fresh, pageKey)
		}
		metrics.Global().RecordPublishSuccess()
		if h.auditLog != nil {
			pubVer := 0
			if fresh != nil {
				pubVer = fresh.PublishedVersion
			}
			h.auditLog.LogPublishSuccess(pageKeyStr, pubVer, user.Username, req.ExpectedDraftVersion)
		}
		invalidateContentCache(h.publicCache, pageKeyStr)
		// also invalidate page caches
		if h.publicCache != nil {
			cache.InvalidatePagePublic(h.publicCache, pageKeyStr)
		}
		publishedAt := time.Now().UTC()
		pubVer := 0
		if fresh != nil {
			pubVer = fresh.PublishedVersion
			if fresh.PublishedAt != nil {
				publishedAt = *fresh.PublishedAt
			}
		}
		c.JSON(http.StatusOK, PublishResponse{
			PageKey:          pageKeyStr,
			PublishedVersion: pubVer,
			PublishedAt:      publishedAt,
		})
		return
	}

	result, err := h.contentSvc.Publish(c.Request.Context(), pageKey, req.ExpectedDraftVersion, user.UserID)
	if err != nil {
		metrics.Global().RecordPublishFailure()
		h.logPublishFail(pageKeyStr, user.Username, err, req.ExpectedDraftVersion)

		if errors.Is(err, service.ErrVersionMismatch) {
			apierror.Write(c, apierror.New(http.StatusConflict, "CONFLICT_VERSION", "Draft version mismatch"))
			return
		}
		if errors.Is(err, service.ErrCannotPublish) {
			apierror.Write(c, apierror.New(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Publish blocked by validation or MediaRef errors"))
			return
		}
		if errors.Is(err, service.ErrDocumentNotFound) {
			apierror.Write(c, apierror.NotFound("Content document not found"))
			return
		}
		apierror.Write(c, apierror.InternalServerError("Failed to publish content"))
		return
	}

	metrics.Global().RecordPublishSuccess()
	if h.auditLog != nil {
		h.auditLog.LogPublishSuccess(pageKeyStr, result.PublishedVersion, user.Username, req.ExpectedDraftVersion)
	}
	invalidateContentCache(h.publicCache, pageKeyStr)

	c.JSON(http.StatusOK, PublishResponse{
		PageKey:          string(result.PageKey),
		PublishedVersion: result.PublishedVersion,
		PublishedAt:      result.PublishedAt,
	})
}

func (h *Handler) logPublishFail(pageKey, actor string, err error, expectedVersion int) {
	if h.auditLog == nil {
		return
	}
	reason := "internal_error"
	details := map[string]interface{}{}
	if errors.Is(err, service.ErrVersionMismatch) {
		reason = "version_mismatch"
		details["expected_version"] = expectedVersion
	} else if errors.Is(err, service.ErrCannotPublish) {
		reason = "validation_failed"
	} else if errors.Is(err, service.ErrDocumentNotFound) {
		reason = "not_found"
	}
	h.auditLog.LogPublishFailure(pageKey, actor, reason, details)
}
