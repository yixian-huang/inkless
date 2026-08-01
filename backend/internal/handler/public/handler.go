package public

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	"github.com/yixian-huang/inkless/backend/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// pageViewTracker is implemented by service.PageViewRecorder.
type pageViewTracker interface {
	Track(pageKey, locale, visitorID, referer string)
}

// Handler handles public content-related HTTP requests
type Handler struct {
	docRepo     repository.ContentDocumentRepository
	pvRepo      repository.PageViewRepository
	pageRepo    repository.UnifiedPageRepository
	cache       *cache.Cache
	viewTracker pageViewTracker
	// legacyFallback merges content_documents when unified has gaps (default true).
	legacyFallback bool
}

// NewHandler creates a new public content handler
func NewHandler(
	docRepo repository.ContentDocumentRepository,
	pvRepo repository.PageViewRepository,
	pageRepo repository.UnifiedPageRepository,
	cache *cache.Cache,
) *Handler {
	return &Handler{
		docRepo:        docRepo,
		pvRepo:         pvRepo,
		pageRepo:       pageRepo,
		cache:          cache,
		legacyFallback: true,
	}
}

// WithViewTracker sets the async page-view recorder.
func (h *Handler) WithViewTracker(t pageViewTracker) *Handler {
	h.viewTracker = t
	return h
}

// WithLegacyContentDocFallback controls dual-track merge from content_documents.
// When false, a complete unified page is returned without a second DB read.
func (h *Handler) WithLegacyContentDocFallback(enabled bool) *Handler {
	h.legacyFallback = enabled
	return h
}

// GetPublicContent handles GET /public/content/{pageKey}?locale=zh|en
// Returns published-only content with locale support.
// Reads from unified_pages first; optionally merges content_documents (legacy).
func (h *Handler) GetPublicContent(c *gin.Context) {
	// Record metrics attempt and start timer
	metrics.Global().RecordPublicGetAttempt()
	startTime := time.Now()

	// Parse page key
	pageKeyStr := c.Param("pageKey")
	pageKey := model.PageKey(pageKeyStr)

	if !pageKey.IsValid() {
		metrics.Global().RecordPublicGetFailure()
		apierror.Write(c, apierror.BadRequest("invalid pageKey"))
		return
	}

	// Parse locale parameter (default to zh)
	locale := c.DefaultQuery("locale", "zh")
	if locale != "zh" && locale != "en" {
		metrics.Global().RecordPublicGetFailure()
		apierror.Write(c, apierror.BadRequest("locale must be zh or en"))
		return
	}

	cacheKey := "content:" + pageKeyStr + ":" + locale
	if cached, ok := h.cache.Get(cacheKey); ok {
		// Count cache hits in success metrics so dashboards reflect real traffic.
		metrics.Global().RecordPublicGetSuccess(time.Since(startTime))
		c.Header("X-Cache", "HIT")
		c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
		c.JSON(200, cached)
		return
	}

	// Try unified_pages first (theme-as-templates: Page is operational SSOT).
	var flatConfig model.JSONMap
	version := 0
	source := "" // page | content_document | merged
	fromPage := false

	if h.pageRepo != nil {
		page, err := h.pageRepo.FindBySlug(c.Request.Context(), pageKeyStr)
		if err == nil && len(page.PublishedConfig) > 0 {
			publishedMap := model.JSONMap(page.PublishedConfig)
			flatConfig = service.ConvertSectionsToContentDoc(pageKeyStr, publishedMap)
			version = page.PublishedVersion
			fromPage = true
			source = "page"
		}
	}

	// Legacy content_documents: always when unified miss; optional merge when
	// unified hit (LEGACY_CONTENT_DOC_FALLBACK, default on).
	needLegacy := flatConfig == nil || h.legacyFallback
	if needLegacy && h.docRepo != nil {
		doc, docErr := h.docRepo.FindByPageKey(c.Request.Context(), pageKey)
		if docErr == nil && len(doc.PublishedConfig) > 0 {
			legacyConfig := model.JSONMap(doc.PublishedConfig)
			if flatConfig == nil {
				flatConfig = legacyConfig
				version = doc.PublishedVersion
				source = "content_document"
			} else {
				// Merge: fill empty keys in flatConfig from legacy
				merged := false
				for k, v := range legacyConfig {
					existing, exists := flatConfig[k]
					if !exists || isEmptyValue(existing) {
						flatConfig[k] = v
						merged = true
					}
				}
				if merged && fromPage {
					source = "merged"
				}
			}
		}
	}

	if flatConfig == nil {
		metrics.Global().RecordPublicGetFailure()
		apierror.Write(c, apierror.NotFound("page not found"))
		return
	}
	if source == "" {
		source = "unknown"
	}

	latency := time.Since(startTime)
	metrics.Global().RecordPublicGetSuccess(latency)

	h.recordPageViewAsync(pageKeyStr, locale, c)

	result := gin.H{
		"pageKey": pageKeyStr,
		"version": version,
		"locale":  locale,
		"config":  flatConfig,
		// source helps agents verify Page dual-read without treating content_documents as SSOT
		"source": source,
	}
	h.cache.Set(cacheKey, result)
	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
	c.JSON(200, result)
}

// isEmptyValue checks if a value is effectively empty (nil, empty map, or empty slice).
func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case map[string]interface{}:
		return len(val) == 0
	case model.JSONMap:
		return len(val) == 0
	case []interface{}:
		return len(val) == 0
	}
	return false
}

// recordPageViewAsync records a page view without blocking the response.
func (h *Handler) recordPageViewAsync(pageKey, locale string, c *gin.Context) {
	clientIP := c.ClientIP()
	referer := c.GetHeader("Referer")
	hash := sha256.Sum256([]byte(clientIP))
	visitorID := fmt.Sprintf("%x", hash[:])[:16]

	if h.viewTracker != nil {
		h.viewTracker.Track(pageKey, locale, visitorID, referer)
		return
	}
	if h.pvRepo == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.pvRepo.Create(ctx, &model.PageView{
			PageKey:   pageKey,
			Locale:    locale,
			VisitorID: visitorID,
			Referer:   referer,
		}); err != nil {
			slog.Error("failed to record page view", "pageKey", pageKey, "error", err)
		}
	}()
}
