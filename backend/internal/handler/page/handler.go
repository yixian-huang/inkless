package page

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles page-related HTTP requests
type Handler struct {
	pageRepo  repository.PageRepository
	themeRepo repository.InstalledThemeRepository
}

// NewHandler creates a new page handler
func NewHandler(pageRepo repository.PageRepository, themeRepo repository.InstalledThemeRepository) *Handler {
	return &Handler{pageRepo: pageRepo, themeRepo: themeRepo}
}

// --- Public endpoints ---

// PublicGetBySlug returns a published page by slug
// GET /public/pages/:slug?locale=zh|en
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	page, err := h.pageRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	if page.Status != model.PageStatusPublished {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	// Build public response with locale filtering
	resp := gin.H{
		"id":       page.ID,
		"slug":     page.Slug,
		"title":    page.Title,
		"template": page.Template,
		"config":   page.Config,
	}

	locale := c.Query("locale")
	if locale == "zh" || locale == "en" {
		resp["title"] = extractLocale(page.Title, locale)
		resp["seoTitle"] = extractLocale(page.SeoTitle, locale)
		resp["seoDescription"] = extractLocale(page.SeoDescription, locale)
	} else {
		resp["seoTitle"] = page.SeoTitle
		resp["seoDescription"] = page.SeoDescription
	}

	c.JSON(http.StatusOK, resp)
}

// PublicList returns all published pages
// GET /public/pages
func (h *Handler) PublicList(c *gin.Context) {
	pages, err := h.pageRepo.ListPublished(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询页面失败"}})
		return
	}

	locale := c.Query("locale")

	items := make([]gin.H, 0, len(pages))
	for _, p := range pages {
		item := gin.H{
			"id":        p.ID,
			"slug":      p.Slug,
			"title":     p.Title,
			"template":  p.Template,
			"sortOrder": p.SortOrder,
			"parentId":  p.ParentID,
		}
		if locale == "zh" || locale == "en" {
			item["title"] = extractLocale(p.Title, locale)
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// PublicListThemePages returns published pages for the active theme
// GET /public/theme-pages
func (h *Handler) PublicListThemePages(c *gin.Context) {
	// Find active theme
	activeTheme, err := h.themeRepo.FindActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
		return
	}

	pages, err := h.pageRepo.ListPublishedByThemeID(c.Request.Context(), activeTheme.ThemeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询页面失败"}})
		return
	}

	items := make([]gin.H, 0, len(pages))
	for _, p := range pages {
		items = append(items, gin.H{
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

	if items == nil {
		items = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// --- Admin endpoints ---

// AdminList returns all pages with optional filters
// GET /admin/pages?status=draft&parentId=1&themeId=corporate-classic
func (h *Handler) AdminList(c *gin.Context) {
	status := c.Query("status")
	themeID := c.Query("themeId")

	// If themeId filter is provided, use theme-specific query
	if themeID != "" {
		pages, err := h.pageRepo.ListByThemeID(c.Request.Context(), themeID, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询页面失败"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": pages})
		return
	}

	var parentID *uint
	if pid := c.Query("parentId"); pid != "" {
		id, err := strconv.ParseUint(pid, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 parentId"}})
			return
		}
		uid := uint(id)
		parentID = &uid
	}

	pages, err := h.pageRepo.List(c.Request.Context(), status, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询页面失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": pages})
}

// AdminGetByID returns a single page by ID
// GET /admin/pages/:id
func (h *Handler) AdminGetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	page, err := h.pageRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	c.JSON(http.StatusOK, page)
}

// createUpdateInput is the JSON body for creating/updating pages
type createUpdateInput struct {
	Slug           string        `json:"slug"`
	ParentID       *uint         `json:"parentId"`
	Title          model.JSONMap `json:"title"`
	Template       string        `json:"template"`
	Config         model.JSONMap `json:"config"`
	Status         string        `json:"status"`
	SortOrder      int           `json:"sortOrder"`
	SeoTitle       model.JSONMap `json:"seoTitle"`
	SeoDescription model.JSONMap `json:"seoDescription"`
	ThemeID        string        `json:"themeId"`
	ContentKey     string        `json:"contentKey"`
	RenderMode     string        `json:"renderMode"`
	IsThemePage    *bool         `json:"isThemePage"`
	NavConfig      model.JSONMap `json:"navConfig"`
	CoverImage     string        `json:"coverImage"`
	AutoSummary    bool          `json:"autoSummary"`
	AllowComments  *bool         `json:"allowComments"`
	Pinned         bool          `json:"pinned"`
	Visibility     string        `json:"visibility"`
	PublishedAt    *string       `json:"publishedAt"`
	Metadata       model.JSONMap `json:"metadata"`
}

// AdminCreate creates a new page
// POST /admin/pages
func (h *Handler) AdminCreate(c *gin.Context) {
	var input createUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	status := model.PageStatus(input.Status)
	if status == "" {
		status = model.PageStatusDraft
	}

	isThemePage := false
	if input.IsThemePage != nil {
		isThemePage = *input.IsThemePage
	}

	allowComments := true
	if input.AllowComments != nil {
		allowComments = *input.AllowComments
	}

	visibility := input.Visibility
	if visibility == "" {
		visibility = "public"
	}

	page := &model.Page{
		Slug:           input.Slug,
		ParentID:       input.ParentID,
		Title:          input.Title,
		Template:       input.Template,
		Config:         input.Config,
		Status:         status,
		SortOrder:      input.SortOrder,
		SeoTitle:       input.SeoTitle,
		SeoDescription: input.SeoDescription,
		ThemeID:        input.ThemeID,
		ContentKey:     input.ContentKey,
		RenderMode:     input.RenderMode,
		IsThemePage:    isThemePage,
		NavConfig:      input.NavConfig,
		CoverImage:     input.CoverImage,
		AutoSummary:    input.AutoSummary,
		AllowComments:  allowComments,
		Pinned:         input.Pinned,
		Visibility:     visibility,
		Metadata:       input.Metadata,
	}

	if input.PublishedAt != nil {
		if t, err := time.Parse(time.RFC3339, *input.PublishedAt); err == nil {
			page.PublishedAt = &t
		}
	}

	if err := h.pageRepo.Create(c.Request.Context(), page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Re-fetch with preloads
	created, err := h.pageRepo.FindByID(c.Request.Context(), page.ID)
	if err != nil {
		c.JSON(http.StatusCreated, page)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// AdminUpdate updates an existing page
// PUT /admin/pages/:id
func (h *Handler) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	existing, err := h.pageRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	var input createUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	existing.Slug = input.Slug
	existing.ParentID = input.ParentID
	existing.Title = input.Title
	existing.Template = input.Template
	existing.Config = input.Config
	existing.SortOrder = input.SortOrder
	existing.SeoTitle = input.SeoTitle
	existing.SeoDescription = input.SeoDescription
	existing.CoverImage = input.CoverImage
	existing.AutoSummary = input.AutoSummary
	existing.Pinned = input.Pinned
	if input.AllowComments != nil {
		existing.AllowComments = *input.AllowComments
	}
	if input.Visibility != "" {
		existing.Visibility = input.Visibility
	}
	if input.Metadata != nil {
		existing.Metadata = input.Metadata
	}
	if input.PublishedAt != nil {
		if t, err := time.Parse(time.RFC3339, *input.PublishedAt); err == nil {
			existing.PublishedAt = &t
		}
	}

	if input.Status != "" {
		existing.Status = model.PageStatus(input.Status)
	}

	// For theme pages, protect contentKey and renderMode
	if !existing.IsThemePage {
		if input.ContentKey != "" {
			existing.ContentKey = input.ContentKey
		}
		if input.RenderMode != "" {
			existing.RenderMode = input.RenderMode
		}
		if input.ThemeID != "" {
			existing.ThemeID = input.ThemeID
		}
	}

	if input.NavConfig != nil {
		existing.NavConfig = input.NavConfig
	}

	if err := h.pageRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Re-fetch with preloads
	updated, err := h.pageRepo.FindByID(c.Request.Context(), existing.ID)
	if err != nil {
		c.JSON(http.StatusOK, existing)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// AdminDelete soft-deletes a page
// DELETE /admin/pages/:id
func (h *Handler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.pageRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// AdminPublish sets page status to published
// PUT /admin/pages/:id/publish
func (h *Handler) AdminPublish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	page, err := h.pageRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	page.Status = model.PageStatusPublished
	if page.PublishedAt == nil {
		now := time.Now()
		page.PublishedAt = &now
	}
	if err := h.pageRepo.Update(c.Request.Context(), page); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "发布失败"}})
		return
	}

	c.JSON(http.StatusOK, page)
}

// AdminUnpublish sets page status to draft
// PUT /admin/pages/:id/unpublish
func (h *Handler) AdminUnpublish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	page, err := h.pageRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "页面不存在"}})
		return
	}

	page.Status = model.PageStatusDraft
	if err := h.pageRepo.Update(c.Request.Context(), page); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "取消发布失败"}})
		return
	}

	c.JSON(http.StatusOK, page)
}

// extractLocale extracts a single locale value from a JSONMap
func extractLocale(m model.JSONMap, locale string) interface{} {
	if m == nil {
		return ""
	}
	if val, ok := m[locale]; ok {
		return val
	}
	// Fallback to zh
	if val, ok := m["zh"]; ok {
		return val
	}
	return ""
}
