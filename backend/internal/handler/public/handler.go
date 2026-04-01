package public

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// Handler handles public content-related HTTP requests
type Handler struct {
	docRepo  repository.ContentDocumentRepository
	pvRepo   repository.PageViewRepository
	pageRepo repository.UnifiedPageRepository
	cache    *cache.Cache
}

// NewHandler creates a new public content handler
func NewHandler(
	docRepo repository.ContentDocumentRepository,
	pvRepo repository.PageViewRepository,
	pageRepo repository.UnifiedPageRepository,
	cache *cache.Cache,
) *Handler {
	return &Handler{
		docRepo:  docRepo,
		pvRepo:   pvRepo,
		pageRepo: pageRepo,
		cache:    cache,
	}
}

// GetPublicContent handles GET /public/content/{pageKey}?locale=zh|en
// Returns published-only content with locale support.
// Reads from unified_pages first (new system); falls back to content_documents (legacy).
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

	cacheKey := "content:" + pageKeyStr + ":" + locale
	if cached, ok := h.cache.Get(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
		c.JSON(200, cached)
		return
	}

	// Try unified_pages first (slug == pageKey for the 7 builtin pages)
	var flatConfig model.JSONMap
	version := 0

	if h.pageRepo != nil {
		page, err := h.pageRepo.FindBySlug(c.Request.Context(), pageKeyStr)
		if err == nil && len(page.PublishedConfig) > 0 {
			publishedMap := model.JSONMap(page.PublishedConfig)
			flatConfig = service.ConvertSectionsToContentDoc(pageKeyStr, publishedMap)
			version = page.PublishedVersion
		}
	}

	// Also read from legacy content_documents to fill gaps (sections migration
	// may have left non-hero props empty).
	doc, docErr := h.docRepo.FindByPageKey(c.Request.Context(), pageKey)
	if docErr == nil && len(doc.PublishedConfig) > 0 {
		legacyConfig := model.JSONMap(doc.PublishedConfig)
		if flatConfig == nil {
			flatConfig = legacyConfig
			version = doc.PublishedVersion
		} else {
			// Merge: fill empty keys in flatConfig from legacy
			for k, v := range legacyConfig {
				existing, exists := flatConfig[k]
				if !exists || isEmptyValue(existing) {
					flatConfig[k] = v
				}
			}
		}
	}

	if flatConfig == nil {
		metrics.Global().RecordPublicGetFailure()
		c.JSON(404, apierror.NotFound("page not found"))
		return
	}

	latency := time.Since(startTime)
	metrics.Global().RecordPublicGetSuccess(latency)

	h.recordPageViewAsync(pageKeyStr, locale, c)

	result := gin.H{
		"pageKey": pageKeyStr,
		"version": version,
		"locale":  locale,
		"config":  flatConfig,
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

// recordPageViewAsync records a page view in a background goroutine.
func (h *Handler) recordPageViewAsync(pageKey, locale string, c *gin.Context) {
	clientIP := c.ClientIP()
	referer := c.GetHeader("Referer")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		hash := sha256.Sum256([]byte(clientIP))
		visitorID := fmt.Sprintf("%x", hash[:])[:16]
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
