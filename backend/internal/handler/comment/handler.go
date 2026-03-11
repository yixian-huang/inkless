package comment

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
)

type Handler struct {
	repo     repository.CommentRepository
	antispam *service.AntiSpamService
}

func NewHandler(repo repository.CommentRepository, antispam *service.AntiSpamService) *Handler {
	return &Handler{repo: repo, antispam: antispam}
}

type createInput struct {
	Content      string `json:"content" binding:"required"`
	AuthorName   string `json:"authorName" binding:"required"`
	AuthorEmail  string `json:"authorEmail"`
	AuthorURL    string `json:"authorUrl"`
	ContentType  string `json:"contentType" binding:"required"`
	ContentID    uint   `json:"contentId" binding:"required"`
	ParentID     *uint  `json:"parentId"`
	CaptchaToken string `json:"captchaToken"`
}

func (h *Handler) PublicCreate(c *gin.Context) {
	var input createInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ip := c.ClientIP()
	if err := h.antispam.Check(c.Request.Context(), ip, input.Content, input.CaptchaToken); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}
	comment := &model.Comment{
		Content:     input.Content,
		AuthorName:  input.AuthorName,
		AuthorEmail: input.AuthorEmail,
		AuthorURL:   input.AuthorURL,
		AuthorIP:    ip,
		ContentType: input.ContentType,
		ContentID:   input.ContentID,
		ParentID:    input.ParentID,
		Status:      model.CommentStatusPending,
	}
	if err := comment.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Create(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func (h *Handler) PublicList(c *gin.Context) {
	contentType := c.Query("contentType")
	contentID, _ := strconv.ParseUint(c.Query("contentId"), 10, 32)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if contentType == "" || contentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentType and contentId required"})
		return
	}
	comments, total, err := h.repo.ListByContent(c.Request.Context(), contentType, uint(contentID), model.CommentStatusApproved, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments, "total": total, "page": page, "pageSize": pageSize})
}

func (h *Handler) AdminList(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	comments, total, err := h.repo.ListAll(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments, "total": total, "page": page, "pageSize": pageSize})
}

func (h *Handler) AdminUpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var input struct {
		Status model.CommentStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.UpdateStatus(c.Request.Context(), uint(id), input.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func (h *Handler) AdminDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) AdminPin(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var input struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SetPinned(c.Request.Context(), uint(id), input.Pinned); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to pin comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *Handler) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.POST("/comments", h.PublicCreate)
	public.GET("/comments", h.PublicList)
	admin.GET("/comments", h.AdminList)
	admin.PATCH("/comments/:id/status", h.AdminUpdateStatus)
	admin.DELETE("/comments/:id", h.AdminDelete)
	admin.PUT("/comments/:id/pin", h.AdminPin)
}
