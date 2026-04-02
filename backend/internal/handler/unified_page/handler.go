package unified_page

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/eventbus"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
)

// Handler handles unified page HTTP requests.
type Handler struct {
	pageRepo    repository.UnifiedPageRepository
	versionRepo repository.PageVersionRepository
	pageSvc     *service.UnifiedPageService
	cache       *cache.Cache
	eventBus    eventbus.EventBus
}

// NewHandler creates a new unified page handler.
func NewHandler(
	pageRepo repository.UnifiedPageRepository,
	versionRepo repository.PageVersionRepository,
	pageSvc *service.UnifiedPageService,
	cache *cache.Cache,
	eventBus eventbus.EventBus,
) *Handler {
	return &Handler{pageRepo: pageRepo, versionRepo: versionRepo, pageSvc: pageSvc, cache: cache, eventBus: eventBus}
}

// getUserID extracts the authenticated user ID from the Gin context.
func getUserID(c *gin.Context) uint {
	uc := middleware.GetUserContext(c)
	if uc == nil {
		return 0
	}
	return uc.UserID
}

// parseID parses the :id URL parameter as uint.
func parseID(c *gin.Context) (uint, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id), true
}

// localizedField returns the zh or en field based on locale query param.
func localizedField(c *gin.Context, zh, en string) string {
	locale := c.Query("locale")
	if locale == "en" && en != "" {
		return en
	}
	return zh
}

// --- Public endpoints ---

// PublicList returns all published unified pages.
func (h *Handler) PublicList(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh")
	cacheKey := "pages:list:" + locale
	if cached, ok := h.cache.Get(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
		c.JSON(http.StatusOK, cached)
		return
	}

	pages, err := h.pageRepo.ListPublished(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pages"})
		return
	}

	items := make([]gin.H, 0, len(pages))
	for _, p := range pages {
		items = append(items, gin.H{
			"id":              p.ID,
			"slug":            p.Slug,
			"title":           localizedField(c, p.ZhTitle, p.EnTitle),
			"description":     localizedField(c, p.ZhDescription, p.EnDescription),
			"mode":            p.Mode,
			"publishedConfig": p.PublishedConfig,
			"sortOrder":       p.SortOrder,
			"showInNav":       p.ShowInNav,
			"parentId":        p.ParentID,
			"metaTitle":       localizedField(c, p.ZhMetaTitle, p.EnMetaTitle),
			"metaDescription": localizedField(c, p.ZhMetaDescription, p.EnMetaDescription),
			"metaKeywords":    localizedField(c, p.ZhMetaKeywords, p.EnMetaKeywords),
		})
	}
	result := gin.H{"items": items}
	h.cache.Set(cacheKey, result)
	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
	c.JSON(http.StatusOK, result)
}

// PublicGetBySlug returns a published unified page by slug.
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	locale := c.DefaultQuery("locale", "zh")

	cacheKey := "page:" + slug + ":" + locale
	if cached, ok := h.cache.Get(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
		c.JSON(http.StatusOK, cached)
		return
	}

	page, err := h.pageRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	if page.PublishedConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	result := gin.H{
		"id":              page.ID,
		"slug":            page.Slug,
		"title":           localizedField(c, page.ZhTitle, page.EnTitle),
		"description":     localizedField(c, page.ZhDescription, page.EnDescription),
		"mode":            page.Mode,
		"publishedConfig": page.PublishedConfig,
		"sortOrder":       page.SortOrder,
		"showInNav":       page.ShowInNav,
		"parentId":        page.ParentID,
		"metaTitle":       localizedField(c, page.ZhMetaTitle, page.EnMetaTitle),
		"metaDescription": localizedField(c, page.ZhMetaDescription, page.EnMetaDescription),
		"metaKeywords":    localizedField(c, page.ZhMetaKeywords, page.EnMetaKeywords),
	}
	h.cache.Set(cacheKey, result)
	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
	c.JSON(http.StatusOK, result)
}

// --- Admin endpoints ---

// AdminList lists all unified pages with optional status/mode/parentID filters.
func (h *Handler) AdminList(c *gin.Context) {
	status := c.Query("status")
	mode := c.Query("mode")

	var parentID *uint
	if pidStr := c.Query("parentId"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 64)
		if err == nil {
			pidVal := uint(pid)
			parentID = &pidVal
		}
	}

	pages, err := h.pageRepo.List(c.Request.Context(), status, mode, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": pages})
}

// AdminGetByID returns a single unified page by ID.
func (h *Handler) AdminGetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	page, err := h.pageRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	c.JSON(http.StatusOK, page)
}

type createInput struct {
	Slug          string    `json:"slug" binding:"required"`
	ZhTitle       string    `json:"zhTitle"`
	EnTitle       string    `json:"enTitle"`
	ZhDescription string    `json:"zhDescription"`
	EnDescription string    `json:"enDescription"`
	Mode          string    `json:"mode"`
	TemplateID    *uint     `json:"templateId"`
	DraftConfig   model.JSONMap `json:"draftConfig"`
	SortOrder     int       `json:"sortOrder"`
	ShowInNav     bool      `json:"showInNav"`
	ParentID      *uint     `json:"parentId"`
	ZhMetaTitle       string `json:"zhMetaTitle"`
	EnMetaTitle       string `json:"enMetaTitle"`
	ZhMetaDescription string `json:"zhMetaDescription"`
	EnMetaDescription string `json:"enMetaDescription"`
	ZhMetaKeywords    string `json:"zhMetaKeywords"`
	EnMetaKeywords    string `json:"enMetaKeywords"`
}

// AdminCreate creates a new unified page.
func (h *Handler) AdminCreate(c *gin.Context) {
	var input createInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page := &model.UnifiedPage{
		Slug:              input.Slug,
		ZhTitle:           input.ZhTitle,
		EnTitle:           input.EnTitle,
		ZhDescription:     input.ZhDescription,
		EnDescription:     input.EnDescription,
		Mode:              input.Mode,
		TemplateID:        input.TemplateID,
		DraftConfig:       input.DraftConfig,
		DraftVersion:      1,
		Status:            "draft",
		SortOrder:         input.SortOrder,
		ShowInNav:         input.ShowInNav,
		ParentID:          input.ParentID,
		ZhMetaTitle:       input.ZhMetaTitle,
		EnMetaTitle:       input.EnMetaTitle,
		ZhMetaDescription: input.ZhMetaDescription,
		EnMetaDescription: input.EnMetaDescription,
		ZhMetaKeywords:    input.ZhMetaKeywords,
		EnMetaKeywords:    input.EnMetaKeywords,
	}

	if err := h.pageRepo.Create(c.Request.Context(), page); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create page: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, page)
}

// AdminGetDraft returns the draft config for a page.
func (h *Handler) AdminGetDraft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	page, err := h.pageRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           page.ID,
		"draftConfig":  page.DraftConfig,
		"draftVersion": page.DraftVersion,
	})
}

type updateDraftInput struct {
	DraftConfig model.JSONMap `json:"draftConfig" binding:"required"`
}

// AdminUpdateDraft updates the draft config with optimistic locking via If-Match header.
func (h *Handler) AdminUpdateDraft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	ifMatch := c.GetHeader("If-Match")
	expectedVersion, err := strconv.Atoi(ifMatch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "If-Match header must be a valid integer version"})
		return
	}

	var input updateDraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newVersion, err := h.pageRepo.UpdateDraft(c.Request.Context(), id, expectedVersion, input.DraftConfig)
	if err != nil {
		// Check for version conflict by re-fetching the page
		page, fetchErr := h.pageRepo.FindByID(c.Request.Context(), id)
		if fetchErr == nil && page.DraftVersion != expectedVersion {
			c.JSON(http.StatusConflict, gin.H{
				"error":          "draft version conflict",
				"currentVersion": page.DraftVersion,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update draft: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           id,
		"draftVersion": newVersion,
	})
}

type publishInput struct {
	ExpectedDraftVersion int `json:"expectedDraftVersion"`
}

// AdminPublish publishes the current draft.
func (h *Handler) AdminPublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var input publishInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if err := h.pageSvc.Publish(c.Request.Context(), id, input.ExpectedDraftVersion, userID); err != nil {
		if errors.Is(err, service.ErrPageVersionConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "draft version conflict"})
			return
		}
		if errors.Is(err, service.ErrUnifiedPageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish: " + err.Error()})
		return
	}

	// Return updated page
	page, err := h.pageRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "published"})
		return
	}

	// Publish content published event
	if h.eventBus != nil {
		h.eventBus.Publish(eventbus.Event{
			Type: eventbus.ContentPublished,
			Payload: eventbus.ContentEventPayload{
				ContentType: "page",
				ContentID:   page.ID,
				Slug:        page.Slug,
				Action:      eventbus.ContentPublished,
			},
		})
	}

	c.JSON(http.StatusOK, page)
}

// AdminUnpublish reverts a page to draft status.
func (h *Handler) AdminUnpublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.pageSvc.Unpublish(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrUnifiedPageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unpublish: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unpublished"})
}

type rollbackInput struct {
	TargetVersion int `json:"targetVersion" binding:"required"`
}

// AdminRollback rolls back to a specific version.
func (h *Handler) AdminRollback(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var input rollbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if err := h.pageSvc.Rollback(c.Request.Context(), id, input.TargetVersion, userID); err != nil {
		if errors.Is(err, service.ErrPageVersionRecNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rollback: " + err.Error()})
		return
	}

	page, err := h.pageRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "rolled back"})
		return
	}
	c.JSON(http.StatusOK, page)
}

// AdminDelete soft-deletes a unified page.
func (h *Handler) AdminDelete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.pageRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete page"})
		return
	}

	// Publish content deleted event
	if h.eventBus != nil {
		h.eventBus.Publish(eventbus.Event{
			Type: eventbus.ContentDeleted,
			Payload: eventbus.ContentEventPayload{
				ContentType: "page",
				ContentID:   uint(id),
				Action:      eventbus.ContentDeleted,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// --- Version history endpoints ---

// AdminListVersions lists version history for a page with pagination.
func (h *Handler) AdminListVersions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	versions, total, err := h.versionRepo.ListByPageID(c.Request.Context(), id, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list versions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    versions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminGetVersionDetail returns a specific version by page ID and version number.
func (h *Handler) AdminGetVersionDetail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	versionNum, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	version, err := h.versionRepo.FindByPageIDAndVersion(c.Request.Context(), id, versionNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}
	c.JSON(http.StatusOK, version)
}
