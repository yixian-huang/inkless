package public

import (
	"context"
	"log/slog"
	"time"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// Handler handles public content-related HTTP requests
type Handler struct {
	docRepo repository.ContentDocumentRepository
	pvRepo  repository.PageViewRepository
}

// NewHandler creates a new public content handler
func NewHandler(docRepo repository.ContentDocumentRepository, pvRepo repository.PageViewRepository) *Handler {
	return &Handler{
		docRepo: docRepo,
		pvRepo:  pvRepo,
	}
}

// GetPublicContent handles GET /public/content/{pageKey}?locale=zh|en
// Returns published-only content with locale support
func (h *Handler) GetPublicContent(c *gin.Context) {
	// Record metrics attempt and start timer
	metrics.Global().RecordPublicGetAttempt()
	startTime := time.Now()

	// Parse page key
	pageKeyStr := c.Param("pageKey")
	pageKey := model.PageKey(pageKeyStr)

	if !pageKey.IsValid() {
		metrics.Global().RecordPublicGetFailure()
		c.JSON(400, apierror.BadRequest("invalid pageKey"))
		return
	}

	// Parse locale parameter (default to zh)
	locale := c.DefaultQuery("locale", "zh")
	if locale != "zh" && locale != "en" {
		metrics.Global().RecordPublicGetFailure()
		c.JSON(400, apierror.BadRequest("locale must be zh or en"))
		return
	}

	// Fetch published content document
	doc, err := h.docRepo.FindByPageKey(c.Request.Context(), pageKey)
	if err != nil {
		metrics.Global().RecordPublicGetFailure()
		c.JSON(404, apierror.NotFound("page not found"))
		return
	}

	// Record success with latency
	latency := time.Since(startTime)
	metrics.Global().RecordPublicGetSuccess(latency)

	// Asynchronously record page view
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.pvRepo.Create(ctx, &model.PageView{
			PageKey: pageKeyStr,
			Locale:  locale,
		}); err != nil {
			slog.Error("failed to record page view", "pageKey", pageKeyStr, "error", err)
		}
	}()

	// Return published-only data (never expose draft fields)
	c.JSON(200, gin.H{
		"pageKey": doc.PageKey.String(),
		"version": doc.PublishedVersion,
		"locale":  locale,
		"config":  doc.PublishedConfig,
	})
}
