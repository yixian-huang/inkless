package global_config

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// Handler serves admin endpoints for site identity / branding config.
// SSOT storage is site_configs key "global". content_documents is legacy read/hydrate only.
type Handler struct {
	siteCfg   repository.SiteConfigRepository
	legacyDoc repository.ContentDocumentRepository // optional hydrate source
	cache     *cache.Cache
}

// NewHandler creates a handler that reads/writes site_configs "global".
func NewHandler(siteCfg repository.SiteConfigRepository, c *cache.Cache) *Handler {
	return &Handler{siteCfg: siteCfg, cache: c}
}

// WithLegacyContentDoc enables one-time hydrate from content_documents.global when
// site_configs has no row yet. Never writes back to content_documents.
func (h *Handler) WithLegacyContentDoc(repo repository.ContentDocumentRepository) *Handler {
	h.legacyDoc = repo
	return h
}

// RegisterRoutes mounts both canonical /site-config and legacy /global-config aliases.
func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	// Canonical path (SSOT naming).
	admin.GET("/site-config", h.adminGet)
	admin.PUT("/site-config/draft", h.adminPutDraft)
	admin.POST("/site-config/publish", h.adminPublish)

	// Legacy alias — same storage; keep for older clients until removed.
	admin.GET("/global-config", h.adminGet)
	admin.PUT("/global-config/draft", h.adminPutDraft)
	admin.POST("/global-config/publish", h.adminPublish)
}

type getResponse struct {
	DraftConfig      model.JSONMap `json:"draftConfig"`
	DraftVersion     int           `json:"draftVersion"`
	PublishedConfig  model.JSONMap `json:"publishedConfig"`
	PublishedVersion int           `json:"publishedVersion"`
	// StorageSource is "site_config", "hydrated_from_content_document", or "empty".
	StorageSource string `json:"storageSource,omitempty"`
}

func (h *Handler) loadSiteGlobal(c *gin.Context) (*model.SiteConfig, string, error) {
	sc, err := h.siteCfg.FindByKey(c.Request.Context(), model.SiteConfigKeyGlobal)
	if err == nil && sc != nil && sc.ID != 0 {
		return sc, "site_config", nil
	}
	// Miss or empty: try one-time hydrate from legacy content_documents.
	if h.legacyDoc != nil {
		hydrated, herr := service.HydrateSiteGlobalFromLegacy(c.Request.Context(), h.siteCfg, h.legacyDoc)
		if herr == nil && hydrated != nil && hydrated.ID != 0 {
			return hydrated, "hydrated_from_content_document", nil
		}
	}
	return nil, "empty", nil
}

func (h *Handler) adminGet(c *gin.Context) {
	sc, source, err := h.loadSiteGlobal(c)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "failed to load site config")
		return
	}
	if sc == nil || sc.ID == 0 {
		c.JSON(http.StatusOK, getResponse{
			DraftConfig:      model.JSONMap{},
			DraftVersion:     0,
			PublishedConfig:  model.JSONMap{},
			PublishedVersion: 0,
			StorageSource:    "empty",
		})
		return
	}
	c.JSON(http.StatusOK, getResponse{
		DraftConfig:      sc.DraftConfig,
		DraftVersion:     sc.DraftVersion,
		PublishedConfig:  sc.PublishedConfig,
		PublishedVersion: sc.PublishedVersion,
		StorageSource:    source,
	})
}

type putDraftInput struct {
	DraftConfig          model.JSONMap `json:"draftConfig"`
	ExpectedDraftVersion int           `json:"expectedDraftVersion"`
}

func (h *Handler) adminPutDraft(c *gin.Context) {
	var input putDraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if _, err := validateGlobalConfig(input.DraftConfig); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure row exists (hydrate legacy if needed).
	sc, _, _ := h.loadSiteGlobal(c)
	if sc == nil || sc.ID == 0 {
		row := &model.SiteConfig{
			Key:              model.SiteConfigKeyGlobal,
			DraftConfig:      input.DraftConfig,
			DraftVersion:     1,
			PublishedConfig:  model.JSONMap{},
			PublishedVersion: 0,
		}
		if err := h.siteCfg.Upsert(c.Request.Context(), row); err != nil {
			apierror.Message(c, http.StatusInternalServerError, "failed to create site config")
			return
		}
		c.JSON(http.StatusOK, gin.H{"draftVersion": 1})
		return
	}

	newVersion, err := h.siteCfg.UpdateDraft(
		c.Request.Context(),
		model.SiteConfigKeyGlobal,
		input.ExpectedDraftVersion,
		input.DraftConfig,
	)
	if err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			apierror.Message(c, http.StatusConflict, "draft version conflict")
			return
		}
		apierror.Message(c, http.StatusInternalServerError, "failed to update draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"draftVersion": newVersion})
}

func (h *Handler) adminPublish(c *gin.Context) {
	sc, _, err := h.loadSiteGlobal(c)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "failed to load site config")
		return
	}
	if sc == nil || sc.ID == 0 {
		apierror.Message(c, http.StatusNotFound, "no draft to publish")
		return
	}
	if _, err := validateGlobalConfig(sc.DraftConfig); err != nil {
		apierror.Message(c, http.StatusBadRequest, "current draft fails validation: "+err.Error())
		return
	}
	newPub := sc.PublishedVersion + 1
	if err := h.siteCfg.UpdatePublished(
		c.Request.Context(),
		model.SiteConfigKeyGlobal,
		sc.DraftConfig,
		newPub,
	); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "failed to publish")
		return
	}
	cache.InvalidateThemeOrSiteConfig(h.cache)
	c.JSON(http.StatusOK, gin.H{"publishedVersion": newPub})
}
