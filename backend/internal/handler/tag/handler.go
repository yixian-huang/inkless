package tag

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles tag-related HTTP requests
type Handler struct {
	tagRepo     repository.TagRepository
	articleRepo repository.ArticleRepository
}

// NewHandler creates a new tag handler
func NewHandler(tagRepo repository.TagRepository, articleRepo repository.ArticleRepository) *Handler {
	return &Handler{tagRepo: tagRepo, articleRepo: articleRepo}
}

// List returns all tags
// GET /admin/tags
func (h *Handler) List(c *gin.Context) {
	items, err := h.tagRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询标签失败"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Create creates a new tag
// POST /admin/tags
func (h *Handler) Create(c *gin.Context) {
	var input model.Tag
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	if err := h.tagRepo.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// Update updates a tag
// PUT /admin/tags/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	existing, err := h.tagRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "标签不存在"}})
		return
	}

	var input model.Tag
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	existing.Slug = input.Slug
	existing.ZhName = input.ZhName
	existing.EnName = input.EnName
	existing.Color = input.Color
	existing.CoverImage = input.CoverImage
	existing.Metadata = input.Metadata

	if err := h.tagRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete deletes a tag
// DELETE /admin/tags/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.tagRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "标签不存在"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// PublicList returns all tags
// GET /public/tags
func (h *Handler) PublicList(c *gin.Context) {
	items, err := h.tagRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询标签失败"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// PublicGetBySlug returns a tag by slug with paginated published articles
// GET /public/tags/:slug?page=1&pageSize=10
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tag, err := h.tagRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "标签不存在"}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	articles, total, err := h.articleRepo.ListPublished(c.Request.Context(), offset, pageSize, "", slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询文章失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tag": tag,
		"articles": gin.H{
			"items":    articles,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
