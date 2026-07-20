package global_config

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// Handler serves admin endpoints for the "global" content document.
type Handler struct {
	repo  repository.ContentDocumentRepository
	cache *cache.Cache
}

func NewHandler(repo repository.ContentDocumentRepository, c *cache.Cache) *Handler {
	return &Handler{repo: repo, cache: c}
}

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	admin.GET("/global-config", h.adminGet)
	admin.PUT("/global-config/draft", h.adminPutDraft)
	admin.POST("/global-config/publish", h.adminPublish)
}

type getResponse struct {
	DraftConfig      model.JSONMap `json:"draftConfig"`
	DraftVersion     int           `json:"draftVersion"`
	PublishedConfig  model.JSONMap `json:"publishedConfig"`
	PublishedVersion int           `json:"publishedVersion"`
}

func (h *Handler) adminGet(c *gin.Context) {
	doc, err := h.repo.FindByPageKey(c.Request.Context(), model.PageKeyGlobal)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "global config not found")
		return
	}
	c.JSON(http.StatusOK, getResponse{
		DraftConfig:      doc.DraftConfig,
		DraftVersion:     doc.DraftVersion,
		PublishedConfig:  doc.PublishedConfig,
		PublishedVersion: doc.PublishedVersion,
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
	newVersion, err := h.repo.UpdateDraft(c.Request.Context(), model.PageKeyGlobal, input.ExpectedDraftVersion, input.DraftConfig)
	if err != nil {
		// Repo returns a string-matched error for version conflict / missing doc.
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
	doc, err := h.repo.FindByPageKey(c.Request.Context(), model.PageKeyGlobal)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "global config not found")
		return
	}
	if _, err := validateGlobalConfig(doc.DraftConfig); err != nil {
		apierror.Message(c, http.StatusBadRequest, "current draft fails validation: "+err.Error())
		return
	}
	newPub := doc.PublishedVersion + 1
	if err := h.repo.UpdatePublished(c.Request.Context(), model.PageKeyGlobal, doc.DraftConfig, newPub); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "failed to publish")
		return
	}
	// Invalidate bootstrap + public content caches for "global".
	cache.InvalidateThemeOrSiteConfig(h.cache)
	c.JSON(http.StatusOK, gin.H{"publishedVersion": newPub})
}
