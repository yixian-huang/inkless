package app

import (
	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/seo"
)

// serveSPAWithMeta renders index.html with SEO meta tags. Returns true if served
// successfully, false if caller should fall back to static file serving.
func serveSPAWithMeta(
	c *gin.Context,
	renderer *seo.Renderer,
	baseURL string,
	siteCfgRepo repository.SiteConfigRepository,
	contentDocRepo repository.ContentDocumentRepository,
) bool {
	if renderer == nil {
		return false
	}
	locale := c.DefaultQuery("locale", "zh")
	if locale != "zh" && locale != "en" {
		locale = "zh"
	}
	meta := seo.ResolveFromPath(c.Request.URL.Path, baseURL, locale)
	if cfg, _, _ := loadSPAGlobalConfig(c, siteCfgRepo, contentDocRepo); len(cfg) > 0 {
		meta.ApplyGlobal(map[string]any(cfg), locale)
	}
	html, err := renderer.Render(meta)
	if err != nil {
		return false
	}
	c.Data(200, "text/html; charset=utf-8", []byte(html))
	c.Abort()
	return true
}

func loadSPAGlobalConfig(
	c *gin.Context,
	siteCfgRepo repository.SiteConfigRepository,
	contentDocRepo repository.ContentDocumentRepository,
) (model.JSONMap, int, string) {
	if siteCfgRepo != nil {
		sc, err := siteCfgRepo.FindByKey(c.Request.Context(), model.SiteConfigKeyGlobal)
		if err == nil && sc != nil && sc.ID != 0 && len(sc.PublishedConfig) > 0 {
			return sc.PublishedConfig, sc.PublishedVersion, "site_config"
		}
	}
	if contentDocRepo != nil {
		if doc, err := contentDocRepo.FindByPageKey(c.Request.Context(), model.PageKeyGlobal); err == nil && doc != nil && len(doc.PublishedConfig) > 0 {
			return doc.PublishedConfig, doc.PublishedVersion, "content_document"
		}
	}
	return model.JSONMap{}, 0, ""
}
