package content

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	"github.com/yixian-huang/inkless/backend/pkg/metrics"
)

// RollbackRequest is POST .../rollback/{version} body.
type RollbackRequest struct {
	ChangeNote string `json:"changeNote"`
}

// RollbackResponse is the rollback success payload.
type RollbackResponse struct {
	PageKey          string    `json:"pageKey"`
	PublishedVersion int       `json:"publishedVersion"`
	SourceVersion    int       `json:"sourceVersion"`
	PublishedAt      time.Time `json:"publishedAt"`
}

// Rollback creates a new published version from a historical snapshot.
func (h *Handler) Rollback(c *gin.Context) {
	pageKeyStr := c.Param("pageKey")
	pageKey := model.PageKey(pageKeyStr)
	metrics.Global().RecordRollbackAttempt()
	startTime := time.Now()

	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	sourceVersion, err := strconv.Atoi(c.Param("version"))
	if err != nil || sourceVersion <= 0 {
		apierror.Write(c, apierror.BadRequest("Invalid version parameter"))
		return
	}

	// Body is optional (changeNote only)
	var req RollbackRequest
	_ = c.ShouldBindJSON(&req)

	user := middleware.GetUserContext(c)
	if user == nil {
		apierror.Write(c, apierror.Unauthorized("User context not found"))
		return
	}

	result, err := h.contentSvc.Rollback(c.Request.Context(), pageKey, sourceVersion, user.UserID)
	if err != nil {
		metrics.Global().RecordRollbackFailure()
		if h.auditLog != nil {
			reason := "internal_error"
			if errors.Is(err, service.ErrVersionNotFound) {
				reason = "version_not_found"
			} else if errors.Is(err, service.ErrDocumentNotFound) {
				reason = "not_found"
			}
			h.auditLog.LogRollbackFailure(pageKeyStr, user.Username, sourceVersion, reason)
		}
		if errors.Is(err, service.ErrVersionNotFound) {
			apierror.Write(c, apierror.NotFound("Source version not found"))
			return
		}
		if errors.Is(err, service.ErrDocumentNotFound) {
			apierror.Write(c, apierror.NotFound("Content document not found"))
			return
		}
		apierror.Write(c, apierror.InternalServerError("Failed to rollback content"))
		return
	}

	metrics.Global().RecordRollbackSuccess(time.Since(startTime))
	if h.auditLog != nil {
		h.auditLog.LogRollbackSuccess(pageKeyStr, result.PublishedVersion, result.SourceVersion, user.Username)
	}
	invalidateContentCache(h.publicCache, pageKeyStr)

	c.JSON(http.StatusOK, RollbackResponse{
		PageKey:          string(result.PageKey),
		PublishedVersion: result.PublishedVersion,
		SourceVersion:    result.SourceVersion,
		PublishedAt:      result.PublishedAt,
	})
}
