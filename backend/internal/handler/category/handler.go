package category

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles category-related HTTP requests
type Handler struct {
	categoryRepo repository.CategoryRepository
	articleRepo  repository.ArticleRepository
}

// NewHandler creates a new category handler
func NewHandler(categoryRepo repository.CategoryRepository, articleRepo repository.ArticleRepository) *Handler {
	return &Handler{categoryRepo: categoryRepo, articleRepo: articleRepo}
}

// List returns all categories
// GET /admin/categories
func (h *Handler) List(c *gin.Context) {
	items, err := h.categoryRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询分类失败"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetByID returns a single category by ID
// GET /admin/categories/:id
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	category, err := h.categoryRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "分类不存在"}})
		return
	}

	c.JSON(http.StatusOK, category)
}

// ListTree returns categories as a tree structure
// GET /admin/categories/tree
func (h *Handler) ListTree(c *gin.Context) {
	items, err := h.categoryRepo.ListTree(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询分类失败"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Create creates a new category
// POST /admin/categories
func (h *Handler) Create(c *gin.Context) {
	var input model.Category
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	if err := h.categoryRepo.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// Update updates a category
// PUT /admin/categories/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	existing, err := h.categoryRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "分类不存在"}})
		return
	}

	var input model.Category
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	existing.Slug = input.Slug
	existing.ZhName = input.ZhName
	existing.EnName = input.EnName
	existing.ParentID = input.ParentID
	existing.CoverImage = input.CoverImage
	existing.ZhDescription = input.ZhDescription
	existing.EnDescription = input.EnDescription
	existing.HideFromList = input.HideFromList
	existing.PreventCascade = input.PreventCascade
	existing.Metadata = input.Metadata
	existing.SortOrder = input.SortOrder

	if err := h.categoryRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete deletes a category
// DELETE /admin/categories/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.categoryRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "分类不存在"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// PublicList returns all visible categories
// GET /public/categories
func (h *Handler) PublicList(c *gin.Context) {
	items, err := h.categoryRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询分类失败"}})
		return
	}

	// Filter out hideFromList categories
	visible := make([]*model.Category, 0, len(items))
	for _, item := range items {
		if !item.HideFromList {
			visible = append(visible, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": visible})
}

// PublicGetBySlug returns a category by slug with paginated published articles
// GET /public/categories/:slug?page=1&pageSize=10
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.categoryRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "分类不存在"}})
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

	articles, total, err := h.articleRepo.ListPublished(c.Request.Context(), offset, pageSize, slug, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询文章失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"articles": gin.H{
			"items":    articles,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
