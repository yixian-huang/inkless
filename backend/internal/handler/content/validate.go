package content

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	"github.com/yixian-huang/inkless/backend/pkg/metrics"
)

// ValidateRequest is POST /admin/content/{pageKey}/validate body.
type ValidateRequest struct {
	Config model.JSONMap `json:"config" binding:"required"`
}

// ValidateResponse is the validation result payload.
type ValidateResponse struct {
	Valid             bool                                `json:"valid"`
	Errors            []service.ValidationError           `json:"errors"`
	TranslationStatus map[string]service.TranslationState `json:"translationStatus"`
	SchemaKind        string                              `json:"schemaKind,omitempty"`
	SchemaID          string                              `json:"schemaId,omitempty"`
	SchemaSource      string                              `json:"schemaSource,omitempty"`
}

// Validate checks a config without saving.
// @Summary      Validate theme content config
// @Description  Schema + MediaRef checks; returns schemaKind (product-first|corporate|…)
// @Tags         Content (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey path string true "Page key"
// @Param        body    body ValidateRequest true "Config to validate"
// @Success      200 {object} ValidateResponse
// @Failure      400 {object} object{error=object}
// @Router       /admin/content/{pageKey}/validate [post]
func (h *Handler) Validate(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	metrics.Global().RecordValidationAttempt()

	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body"))
		return
	}
	if req.Config == nil {
		req.Config = model.JSONMap{}
	}

	actorName := "unknown"
	if user := middleware.GetUserContext(c); user != nil {
		actorName = user.Username
	}

	var result *service.ValidationResult
	if h.slots != nil {
		res, slot, ok := h.slots.ResolveSlot(c.Request.Context(), string(pageKey))
		if ok {
			s := slot
			result = h.validationSvc.ValidateConfigWithSlot(pageKey, req.Config, &s, "theme")
		} else {
			src := res.Source
			if src == "" || src == "none" {
				src = "host-fallback"
			}
			result = h.validationSvc.ValidateConfigWithSlot(pageKey, req.Config, nil, src)
		}
	} else {
		result = h.validationSvc.ValidateConfig(pageKey, req.Config)
	}
	if !result.Valid {
		metrics.Global().RecordValidationFailure()
	}

	translationIssueCount := 0
	for _, state := range result.TranslationStatus {
		if state == service.TranslationStateMissing || state == service.TranslationStateStale {
			translationIssueCount++
		}
	}
	if h.auditLog != nil {
		h.auditLog.LogValidation(string(pageKey), actorName, result.Valid, len(result.Errors), translationIssueCount)
	}

	c.JSON(http.StatusOK, ValidateResponse{
		Valid:             result.Valid,
		Errors:            result.Errors,
		TranslationStatus: result.TranslationStatus,
		SchemaKind:        result.SchemaKind,
		SchemaID:          result.SchemaID,
		SchemaSource:      result.SchemaSource,
	})
}
