package bootstrap

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler aggregates multiple public endpoints into a single response
// to minimize round-trips for SPA initial load.
type Handler struct {
	contentDocRepo repository.ContentDocumentRepository
	themeRepo      repository.InstalledThemeRepository
	pageRepo       repository.PageRepository
}

// NewHandler creates a new bootstrap handler
func NewHandler(
	contentDocRepo repository.ContentDocumentRepository,
	themeRepo repository.InstalledThemeRepository,
	pageRepo repository.PageRepository,
) *Handler {
	return &Handler{
		contentDocRepo: contentDocRepo,
		themeRepo:      themeRepo,
		pageRepo:       pageRepo,
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
			"maxWidth":        "1200px",
			"borderRadius":    "0.5rem",
			"contentPadding":  "1.5rem",
			"sectionSpacing":  "5rem",
			"contentGap":      "2rem",
		},
	}
}

// PublicBootstrap returns all data needed for SPA initial render in one response.
// GET /public/bootstrap?locale=zh|en&pageKey=home
func (h *Handler) PublicBootstrap(c *gin.Context) {
	ctx := c.Request.Context()
	locale := c.DefaultQuery("locale", "zh")
	pageKey := c.Query("pageKey")

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

	// 2. Theme tokens
	var themeTokens interface{}
	doc, err := h.contentDocRepo.FindByPageKey(ctx, model.PageKeyTheme)
	if err != nil || len(doc.PublishedConfig) == 0 {
		themeTokens = defaultThemeConfig()
	} else {
		themeTokens = doc.PublishedConfig
	}

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

	// 4. Global config
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

	// 5. Page content (optional, only if pageKey is provided)
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
		"globalConfig": globalConfig,
	}

	if pageContent != nil {
		result["pageContent"] = pageContent
	}

	c.JSON(http.StatusOK, result)
}
