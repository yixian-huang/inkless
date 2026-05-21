package features

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler serves admin endpoints for the site_configs "features" key.
type Handler struct {
	repo  repository.SiteConfigRepository
	cache *cache.Cache
}

func NewHandler(repo repository.SiteConfigRepository, c *cache.Cache) *Handler {
	return &Handler{repo: repo, cache: c}
}

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	admin.GET("/features", h.adminGet)
	admin.PUT("/features/draft", h.adminPutDraft)
	admin.POST("/features/publish", h.adminPublish)
}

func (h *Handler) adminGet(c *gin.Context) {
	sc, err := h.repo.FindByKey(c.Request.Context(), model.SiteConfigKeyFeatures)
	if err != nil || sc == nil || sc.ID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"draftConfig":      model.JSONMap{},
			"draftVersion":     0,
			"publishedConfig":  model.JSONMap{},
			"publishedVersion": 0,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"draftConfig":      sc.DraftConfig,
		"draftVersion":     sc.DraftVersion,
		"publishedConfig":  sc.PublishedConfig,
		"publishedVersion": sc.PublishedVersion,
	})
}

type putDraftInput struct {
	DraftConfig          model.JSONMap `json:"draftConfig"`
	ExpectedDraftVersion int           `json:"expectedDraftVersion"`
}

func (h *Handler) adminPutDraft(c *gin.Context) {
	var in putDraftInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid body"}})
		return
	}
	if _, err := validateFeatures(in.DraftConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	existing, ferr := h.repo.FindByKey(c.Request.Context(), model.SiteConfigKeyFeatures)
	if ferr != nil || existing == nil || existing.ID == 0 {
		sc := &model.SiteConfig{
			Key:              model.SiteConfigKeyFeatures,
			DraftConfig:      in.DraftConfig,
			DraftVersion:     1,
			PublishedConfig:  model.JSONMap{},
			PublishedVersion: 0,
		}
		if err := h.repo.Upsert(c.Request.Context(), sc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "create failed"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"draftVersion": 1})
		return
	}
	newVersion, err := h.repo.UpdateDraft(c.Request.Context(), model.SiteConfigKeyFeatures, in.ExpectedDraftVersion, in.DraftConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "update failed"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draftVersion": newVersion})
}

func (h *Handler) adminPublish(c *gin.Context) {
	sc, err := h.repo.FindByKey(c.Request.Context(), model.SiteConfigKeyFeatures)
	if err != nil || sc == nil || sc.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "no draft to publish"}})
		return
	}
	if _, err := validateFeatures(sc.DraftConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	newPub := sc.PublishedVersion + 1
	if err := h.repo.UpdatePublished(c.Request.Context(), model.SiteConfigKeyFeatures, sc.DraftConfig, newPub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "publish failed"}})
		return
	}
	if h.cache != nil {
		h.cache.Flush()
	}
	c.JSON(http.StatusOK, gin.H{"publishedVersion": newPub})
}
