package article

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles article-related HTTP requests
type Handler struct {
	articleRepo  repository.ArticleRepository
	categoryRepo repository.CategoryRepository
	tagRepo      repository.TagRepository
}

// NewHandler creates a new article handler
func NewHandler(
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	tagRepo repository.TagRepository,
) *Handler {
	return &Handler{
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
	}
}

// --- Public endpoints ---

// PublicList returns a paginated list of published articles
// GET /public/articles?page=1&pageSize=10&category=&tag=
func (h *Handler) PublicList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	categorySlug := c.Query("category")
	tagSlug := c.Query("tag")
	offset := (page - 1) * pageSize

	items, total, err := h.articleRepo.ListPublished(c.Request.Context(), offset, pageSize, categorySlug, tagSlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询文章失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// PublicGetBySlug returns a single published article by slug
// GET /public/articles/:slug
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	article, err := h.articleRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "文章不存在"}})
		return
	}

	if article.Status != model.ArticleStatusPublished {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "文章不存在"}})
		return
	}

	// Only return publicly visible articles
	if article.Visibility != "" && article.Visibility != "public" {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "文章不存在"}})
		return
	}

	c.JSON(http.StatusOK, article)
}

// --- Admin endpoints ---

// AdminList returns a paginated list of articles (all statuses)
// GET /admin/articles?page=1&pageSize=10&status=
func (h *Handler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	status := c.Query("status")
	offset := (page - 1) * pageSize

	items, total, err := h.articleRepo.List(c.Request.Context(), offset, pageSize, status, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询文章失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminGetByID returns a single article by ID
// GET /admin/articles/:id
func (h *Handler) AdminGetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	article, err := h.articleRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "文章不存在"}})
		return
	}

	c.JSON(http.StatusOK, article)
}

// createUpdateInput is the JSON body for creating/updating articles
type createUpdateInput struct {
	Slug              string        `json:"slug"`
	Status            string        `json:"status"`
	ZhTitle           string        `json:"zhTitle"`
	EnTitle           string        `json:"enTitle"`
	ZhBody            string        `json:"zhBody"`
	EnBody            string        `json:"enBody"`
	CoverImage        string        `json:"coverImage"`
	ZhSeoTitle        string        `json:"zhSeoTitle"`
	EnSeoTitle        string        `json:"enSeoTitle"`
	ZhMetaDescription string        `json:"zhMetaDescription"`
	EnMetaDescription string        `json:"enMetaDescription"`
	OgImage           string        `json:"ogImage"`
	CategoryID        *uint         `json:"categoryId"`
	CategoryIDs       []uint        `json:"categoryIds"`
	TagIDs            []uint        `json:"tagIds"`
	Author            string        `json:"author"`
	AutoSummary       bool          `json:"autoSummary"`
	AllowComments     *bool         `json:"allowComments"`
	Pinned            bool          `json:"pinned"`
	Visibility        string        `json:"visibility"`
	Metadata          model.JSONMap `json:"metadata"`
}

// AdminCreate creates a new article
// POST /admin/articles
func (h *Handler) AdminCreate(c *gin.Context) {
	var input createUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	status := model.ArticleStatus(input.Status)
	if status == "" {
		status = model.ArticleStatusDraft
	}

	allowComments := true
	if input.AllowComments != nil {
		allowComments = *input.AllowComments
	}

	visibility := input.Visibility
	if visibility == "" {
		visibility = "public"
	}

	article := &model.Article{
		Slug:              input.Slug,
		Status:            status,
		ZhTitle:           input.ZhTitle,
		EnTitle:           input.EnTitle,
		ZhBody:            input.ZhBody,
		EnBody:            input.EnBody,
		CoverImage:        input.CoverImage,
		ZhSeoTitle:        input.ZhSeoTitle,
		EnSeoTitle:        input.EnSeoTitle,
		ZhMetaDescription: input.ZhMetaDescription,
		EnMetaDescription: input.EnMetaDescription,
		OgImage:           input.OgImage,
		CategoryID:        input.CategoryID,
		Author:            input.Author,
		AutoSummary:       input.AutoSummary,
		AllowComments:     allowComments,
		Pinned:            input.Pinned,
		Visibility:        visibility,
		Metadata:          input.Metadata,
	}

	if status == model.ArticleStatusPublished {
		now := time.Now()
		article.PublishedAt = &now
	}

	// Resolve categories
	if len(input.CategoryIDs) > 0 {
		categories, err := h.resolveCategoryIDs(c, input.CategoryIDs)
		if err != nil {
			return
		}
		article.Categories = categories
	}

	// Resolve tags
	if len(input.TagIDs) > 0 {
		tags, err := h.resolveTagIDs(c, input.TagIDs)
		if err != nil {
			return // error already written to response
		}
		article.Tags = tags
	}

	if err := h.articleRepo.Create(c.Request.Context(), article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Re-fetch with preloads
	created, err := h.articleRepo.FindByID(c.Request.Context(), article.ID)
	if err != nil {
		c.JSON(http.StatusCreated, article)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// AdminUpdate updates an existing article
// PUT /admin/articles/:id
func (h *Handler) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	existing, err := h.articleRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "文章不存在"}})
		return
	}

	var input createUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	// Update fields
	existing.Slug = input.Slug
	existing.ZhTitle = input.ZhTitle
	existing.EnTitle = input.EnTitle
	existing.ZhBody = input.ZhBody
	existing.EnBody = input.EnBody
	existing.CoverImage = input.CoverImage
	existing.ZhSeoTitle = input.ZhSeoTitle
	existing.EnSeoTitle = input.EnSeoTitle
	existing.ZhMetaDescription = input.ZhMetaDescription
	existing.EnMetaDescription = input.EnMetaDescription
	existing.OgImage = input.OgImage
	existing.CategoryID = input.CategoryID
	existing.Author = input.Author
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

	if input.Status != "" {
		newStatus := model.ArticleStatus(input.Status)
		// Set publishedAt when transitioning to published
		if newStatus == model.ArticleStatusPublished && existing.Status != model.ArticleStatusPublished {
			now := time.Now()
			existing.PublishedAt = &now
		}
		existing.Status = newStatus
	}

	// Resolve categories: prefer CategoryIDs, fallback to CategoryID
	if input.CategoryIDs != nil {
		categories, err := h.resolveCategoryIDs(c, input.CategoryIDs)
		if err != nil {
			return
		}
		existing.Categories = categories
	}

	// Resolve tags
	if input.TagIDs != nil {
		tags, err := h.resolveTagIDs(c, input.TagIDs)
		if err != nil {
			return
		}
		existing.Tags = tags
	}

	if err := h.articleRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Re-fetch with preloads
	updated, err := h.articleRepo.FindByID(c.Request.Context(), existing.ID)
	if err != nil {
		c.JSON(http.StatusOK, existing)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// AdminDelete deletes an article
// DELETE /admin/articles/:id
func (h *Handler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.articleRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "文章不存在"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// resolveCategoryIDs looks up categories by their IDs and returns them
func (h *Handler) resolveCategoryIDs(c *gin.Context, categoryIDs []uint) ([]model.Category, error) {
	categories, err := h.categoryRepo.FindByIDs(c.Request.Context(), categoryIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询分类失败"}})
		return nil, err
	}
	if len(categories) != len(categoryIDs) {
		found := make(map[uint]bool, len(categories))
		for _, cat := range categories {
			found[cat.ID] = true
		}
		for _, id := range categoryIDs {
			if !found[id] {
				c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "分类 ID " + strconv.FormatUint(uint64(id), 10) + " 不存在"}})
				return nil, fmt.Errorf("category %d not found", id)
			}
		}
	}
	return categories, nil
}

// resolveTagIDs looks up tags by their IDs and returns them
func (h *Handler) resolveTagIDs(c *gin.Context, tagIDs []uint) ([]model.Tag, error) {
	tags, err := h.tagRepo.FindByIDs(c.Request.Context(), tagIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询标签失败"}})
		return nil, err
	}
	if len(tags) != len(tagIDs) {
		found := make(map[uint]bool, len(tags))
		for _, tag := range tags {
			found[tag.ID] = true
		}
		for _, id := range tagIDs {
			if !found[id] {
				c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "标签 ID " + strconv.FormatUint(uint64(id), 10) + " 不存在"}})
				return nil, fmt.Errorf("tag %d not found", id)
			}
		}
	}
	return tags, nil
}
