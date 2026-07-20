package category

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/contentexcerpt"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
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

// List returns all categories.
// @Summary      List all categories
// @Description  Returns all categories for admin management
// @Tags         Categories (Admin)
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{items=[]object}
// @Router       /admin/categories [get]
func (h *Handler) List(c *gin.Context) {
	items, err := h.categoryRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询分类失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetByID returns a single category by ID
// GET /admin/categories/:id
// @Summary      Get category by ID
// @Description  Returns a single category by its database ID
// @Tags         Categories (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Category ID"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/categories/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	category, err := h.categoryRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "分类不存在")
		return
	}

	c.JSON(http.StatusOK, category)
}

// ListTree returns categories as a tree structure
// GET /admin/categories/tree
// @Summary      List categories as tree
// @Description  Returns categories organized in a parent-child tree structure
// @Tags         Categories (Admin)
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{items=[]object}
// @Router       /admin/categories/tree [get]
func (h *Handler) ListTree(c *gin.Context) {
	items, err := h.categoryRepo.ListTree(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询分类失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Create creates a new category.
// @Summary      Create category
// @Description  Create a new category
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Category data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/categories [post]
func (h *Handler) Create(c *gin.Context) {
	var input model.Category
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if err := h.categoryRepo.Create(c.Request.Context(), &input); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, input)
}

// Update updates a category.
// @Summary      Update category
// @Description  Update an existing category by ID
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Category ID"
// @Param        body body object true "Updated category data"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/categories/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	existing, err := h.categoryRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "分类不存在")
		return
	}

	var input model.Category
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
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
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete deletes a category.
// @Summary      Delete category
// @Description  Delete a category by ID
// @Tags         Categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Category ID"
// @Success      200 {object} object{message=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/categories/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.categoryRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "分类不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// PublicList returns all visible categories
// GET /public/categories
// @Summary      List visible categories
// @Description  Returns all categories that are not hidden from public listing
// @Tags         Categories
// @Produce      json
// @Success      200 {object} object{items=[]object}
// @Router       /public/categories [get]
func (h *Handler) PublicList(c *gin.Context) {
	items, err := h.categoryRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询分类失败")
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
// @Summary      Get category by slug
// @Description  Returns a category by slug with paginated published articles
// @Tags         Categories
// @Produce      json
// @Param        slug     path  string true  "Category slug"
// @Param        page     query int    false "Page number"    default(1)
// @Param        pageSize query int    false "Items per page" default(10)
// @Success      200 {object} object{category=object,articles=object}
// @Failure      404 {object} object{error=string}
// @Router       /public/categories/{slug} [get]
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.categoryRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "分类不存在")
		return
	}

	p := handlerutil.ParsePagination(c, 10, 100)
	page, pageSize := p.Page, p.PageSize
	offset := p.Offset

	articles, total, err := h.articleRepo.ListPublished(c.Request.Context(), offset, pageSize, slug, "")
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询文章失败")
		return
	}
	contentexcerpt.ApplyListExcerpts(articles)

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
