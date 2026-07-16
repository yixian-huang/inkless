package bootstrap

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	featurespkg "blotting-consultancy/internal/handler/features"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler aggregates multiple public endpoints into a single response
// to minimize round-trips for SPA initial load.
type Handler struct {
	contentDocRepo  repository.ContentDocumentRepository
	themeRepo       repository.InstalledThemeRepository
	pageRepo        repository.PageRepository
	unifiedPageRepo repository.UnifiedPageRepository
	siteCfgRepo     repository.SiteConfigRepository
	cache           *cache.Cache
}

// NewHandler creates a new bootstrap handler
func NewHandler(
	contentDocRepo repository.ContentDocumentRepository,
	themeRepo repository.InstalledThemeRepository,
	pageRepo repository.PageRepository,
	unifiedPageRepo repository.UnifiedPageRepository,
	siteCfgRepo repository.SiteConfigRepository,
	cache *cache.Cache,
) *Handler {
	return &Handler{
		contentDocRepo:  contentDocRepo,
		themeRepo:       themeRepo,
		pageRepo:        pageRepo,
		unifiedPageRepo: unifiedPageRepo,
		siteCfgRepo:     siteCfgRepo,
		cache:           cache,
	}
}

// defaultThemeConfig returns the default theme token values (same as theme handler)
func defaultThemeConfig() model.JSONMap {
	return model.JSONMap{
		"colors": map[string]interface{}{
			"primary":     "#1a5f8f",
			"primaryDark": "#26548b",
			"accent":      "#8bc34a",
			"accentHover": "#7cb342",
			"surface":     "#ffffff",
			"surfaceAlt":  "#f9fafb",
		},
		"fonts": map[string]interface{}{
			"sans":    "system-ui, -apple-system, sans-serif",
			"heading": "system-ui, -apple-system, sans-serif",
		},
		"layout": map[string]interface{}{
			"maxWidth":       "1200px",
			"borderRadius":   "0.5rem",
			"contentPadding": "1.5rem",
			"sectionSpacing": "5rem",
			"contentGap":     "2rem",
		},
	}
}

// PublicBootstrap returns all data needed for SPA initial render.
// @Summary      Bootstrap SPA
// @Description  Returns active theme, tokens, pages, global config in a single response
// @Tags         Bootstrap
// @Produce      json
// @Param        locale  query string false "Locale (zh or en)" default(zh)
// @Param        pageKey query string false "Optional page key to include content"
// @Success      200 {object} object{activeTheme=object,themeTokens=object,themePages=[]object,unifiedPages=[]object,globalConfig=object}
// @Router       /public/bootstrap [get]
func (h *Handler) PublicBootstrap(c *gin.Context) {
	ctx := c.Request.Context()
	locale := c.DefaultQuery("locale", "zh")
	pageKey := c.Query("pageKey")

	cacheKey := "bootstrap:" + locale + ":" + pageKey
	if cached, ok := h.cache.Get(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Header("Cache-Control", "private, no-cache, must-revalidate")
		c.JSON(http.StatusOK, cached)
		return
	}

	// 1. Active theme
	var activeThemeData gin.H
	var activeThemeID string
	activeTheme, err := h.themeRepo.FindActive(ctx)
	if err != nil {
		activeThemeData = gin.H{}
	} else {
		activeThemeID = activeTheme.ThemeID
		activeThemeData = gin.H{
			"themeId":     activeTheme.ThemeID,
			"source":      activeTheme.Source,
			"externalUrl": activeTheme.ExternalURL,
			"config":      activeTheme.Config,
		}
	}

	// 2. Theme tokens — site_configs "theme" is canonical; content_documents "theme" is legacy fallback
	themeTokens := h.loadPublishedThemeTokens(ctx)

	// 3. Theme pages (only if we have an active theme)
	var themePages []gin.H
	if activeThemeID != "" {
		pages, err := h.pageRepo.ListPublishedByThemeID(ctx, activeThemeID)
		if err == nil {
			for _, p := range pages {
				themePages = append(themePages, gin.H{
					"id":          p.ID,
					"slug":        p.Slug,
					"title":       p.Title,
					"contentKey":  p.ContentKey,
					"renderMode":  p.RenderMode,
					"isThemePage": p.IsThemePage,
					"themeId":     p.ThemeID,
					"navConfig":   p.NavConfig,
					"sortOrder":   p.SortOrder,
					"status":      p.Status,
				})
			}
		}
	}
	if themePages == nil {
		themePages = []gin.H{}
	}

	// 4. Published unified pages are the editable source of truth for public
	// routes and automatic navigation. Theme pages remain a compatibility and
	// rendering-component source for older installations.
	unifiedPages := make([]gin.H, 0)
	if h.unifiedPageRepo != nil {
		pages, err := h.unifiedPageRepo.ListPublished(ctx)
		if err == nil {
			for _, p := range pages {
				if len(p.PublishedConfig) == 0 {
					continue
				}
				unifiedPages = append(unifiedPages, gin.H{
					"id":               p.ID,
					"slug":             p.Slug,
					"title":            gin.H{"zh": p.ZhTitle, "en": p.EnTitle},
					"description":      gin.H{"zh": p.ZhDescription, "en": p.EnDescription},
					"mode":             p.Mode,
					"sortOrder":        p.SortOrder,
					"showInNav":        p.ShowInNav,
					"parentId":         p.ParentID,
					"status":           p.Status,
					"publishedVersion": p.PublishedVersion,
				})
			}
		}
	}

	// 5. Global config
	var globalConfig interface{}
	globalDoc, err := h.contentDocRepo.FindByPageKey(ctx, model.PageKey("global"))
	if err != nil {
		globalConfig = gin.H{}
	} else {
		globalConfig = gin.H{
			"pageKey": globalDoc.PageKey.String(),
			"version": globalDoc.PublishedVersion,
			"locale":  locale,
			"config":  globalDoc.PublishedConfig,
		}
	}

	// 6. Features config
	var features interface{}
	featuresCfg, err := h.siteCfgRepo.FindByKey(ctx, model.SiteConfigKeyFeatures)
	if err != nil || featuresCfg == nil || featuresCfg.ID == 0 || featuresCfg.PublishedConfig == nil {
		features = gin.H{}
	} else {
		features = featurespkg.MergePublishedDefaults(featuresCfg.PublishedConfig)
	}

	// 7. Page content (optional, only if pageKey is provided)
	var pageContent interface{}
	if pageKey != "" {
		pk := model.PageKey(pageKey)
		if pk.IsValid() {
			pageDoc, err := h.contentDocRepo.FindByPageKey(ctx, pk)
			if err == nil {
				pageContent = gin.H{
					"pageKey": pageDoc.PageKey.String(),
					"version": pageDoc.PublishedVersion,
					"locale":  locale,
					"config":  pageDoc.PublishedConfig,
				}
			}
		}
	}

	result := gin.H{
		"activeTheme":  activeThemeData,
		"themeTokens":  themeTokens,
		"themePages":   themePages,
		"unifiedPages": unifiedPages,
		"globalConfig": globalConfig,
		"features":     features,
	}

	if pageContent != nil {
		result["pageContent"] = pageContent
	}

	h.cache.Set(cacheKey, result)
	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", "private, no-cache, must-revalidate")
	c.JSON(http.StatusOK, result)
}

// loadPublishedThemeTokens reads design tokens from site_configs, with legacy fallback.
func (h *Handler) loadPublishedThemeTokens(ctx context.Context) interface{} {
	if sc, err := h.siteCfgRepo.FindByKey(ctx, model.SiteConfigKeyTheme); err == nil && sc != nil && len(sc.PublishedConfig) > 0 {
		return sc.PublishedConfig
	}
	if doc, err := h.contentDocRepo.FindByPageKey(ctx, model.PageKeyTheme); err == nil && len(doc.PublishedConfig) > 0 {
		return doc.PublishedConfig
	}
	return defaultThemeConfig()
}
