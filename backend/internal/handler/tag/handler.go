package tag

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/contentexcerpt"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
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
// @Summary      List all tags
// @Description  Returns all tags for admin management
// @Tags         Tags (Admin)
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{items=[]object}
// @Router       /admin/tags [get]
func (h *Handler) List(c *gin.Context) {
	items, err := h.tagRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询标签失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Create creates a new tag
// POST /admin/tags
// @Summary      Create tag
// @Description  Create a new tag
// @Tags         Tags (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Tag data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/tags [post]
func (h *Handler) Create(c *gin.Context) {
	var input model.Tag
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if err := h.tagRepo.Create(c.Request.Context(), &input); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, input)
}

// Update updates a tag
// PUT /admin/tags/:id
// @Summary      Update tag
// @Description  Update an existing tag by ID
// @Tags         Tags (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Tag ID"
// @Param        body body object true "Updated tag data"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/tags/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	existing, err := h.tagRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "标签不存在")
		return
	}

	var input model.Tag
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	existing.Slug = input.Slug
	existing.ZhName = input.ZhName
	existing.EnName = input.EnName
	existing.Color = input.Color
	existing.CoverImage = input.CoverImage
	existing.Metadata = input.Metadata

	if err := h.tagRepo.Update(c.Request.Context(), existing); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete deletes a tag
// DELETE /admin/tags/:id
// @Summary      Delete tag
// @Description  Delete a tag by ID
// @Tags         Tags (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Tag ID"
// @Success      200 {object} object{message=string}
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/tags/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.tagRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "标签不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// PublicList returns all tags
// GET /public/tags
// @Summary      List all tags
// @Description  Returns all tags for public display
// @Tags         Tags
// @Produce      json
// @Success      200 {object} object{items=[]object}
// @Router       /public/tags [get]
func (h *Handler) PublicList(c *gin.Context) {
	items, err := h.tagRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询标签失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// PublicGetBySlug returns a tag by slug with paginated published articles
// GET /public/tags/:slug?page=1&pageSize=10
// @Summary      Get tag by slug
// @Description  Returns a tag by slug with paginated published articles
// @Tags         Tags
// @Produce      json
// @Param        slug     path  string true  "Tag slug"
// @Param        page     query int    false "Page number"    default(1)
// @Param        pageSize query int    false "Items per page" default(10)
// @Success      200 {object} object{tag=object,articles=object}
// @Failure      404 {object} object{error=string}
// @Router       /public/tags/{slug} [get]
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tag, err := h.tagRepo.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "标签不存在")
		return
	}

	p := handlerutil.ParsePagination(c, 10, 100)
	page, pageSize := p.Page, p.PageSize
	offset := p.Offset

	articles, total, err := h.articleRepo.ListPublished(c.Request.Context(), offset, pageSize, "", slug)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询文章失败")
		return
	}
	contentexcerpt.ApplyListExcerpts(articles)

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
