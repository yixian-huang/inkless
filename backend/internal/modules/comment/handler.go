package comment

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// Handler handles comment HTTP requests.
type Handler struct {
	repo           Repository
	antispam       *AntiSpamService
	siteCfgRepo    repository.SiteConfigRepository
	contentDocRepo repository.ContentDocumentRepository
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

// featureEnabled checks if the comment feature is enabled in SiteConfig.
func (h *Handler) featureEnabled(c *gin.Context) bool {
	cfg, err := h.siteCfgRepo.FindByKey(c.Request.Context(), model.SiteConfigKeyFeatures)
	if err != nil || cfg == nil {
		// Default to enabled if no config exists
		return true
	}
	if cfg.PublishedConfig == nil {
		return true
	}
	commentVal, ok := cfg.PublishedConfig["comment"]
	if !ok {
		return true
	}
	commentMap, ok := commentVal.(map[string]interface{})
	if !ok {
		return true
	}
	enabled, ok := commentMap["enabled"]
	if !ok {
		return true
	}
	b, ok := enabled.(bool)
	return !ok || b
}

// PublicCreate creates a new comment.
// @Summary      Create comment
// @Description  Submit a new comment on content (subject to anti-spam checks)
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Param        body body createInput true "Comment data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      429 {object} object{error=string}
// @Router       /public/comments [post]
func (h *Handler) PublicCreate(c *gin.Context) {
	if !h.featureEnabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "not found"}})
		return
	}

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
	comment := &Comment{
		Content:     input.Content,
		AuthorName:  input.AuthorName,
		AuthorEmail: input.AuthorEmail,
		AuthorURL:   input.AuthorURL,
		AuthorIP:    ip,
		ContentType: input.ContentType,
		ContentID:   input.ContentID,
		ParentID:    input.ParentID,
		AuthorRole:  AuthorRoleGuest,
		Status:      CommentStatusPending,
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

// PublicList returns approved comments for content.
// @Summary      List comments
// @Description  Returns paginated approved comments for a given content item
// @Tags         Comments
// @Produce      json
// @Param        contentType query string true  "Content type"
// @Param        contentId   query int    true  "Content ID"
// @Param        page        query int    false "Page number"    default(1)
// @Param        pageSize    query int    false "Items per page" default(20)
// @Success      200 {object} object{comments=[]object,total=int}
// @Router       /public/comments [get]
func (h *Handler) PublicList(c *gin.Context) {
	contentType := c.Query("contentType")
	contentID, _ := strconv.ParseUint(c.Query("contentId"), 10, 32)
	p := handlerutil.ParsePagination(c, 20, handlerutil.DefaultMaxPageSize)
	if contentType == "" || contentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentType and contentId required"})
		return
	}
	comments, total, err := h.repo.ListByContent(c.Request.Context(), contentType, uint(contentID), CommentStatusApproved, p.Page, p.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments, "total": total, "page": p.Page, "pageSize": p.PageSize})
}

// AdminList returns all comments with optional status filter.
// @Summary      List all comments (admin)
// @Description  Returns paginated comments with optional status filtering
// @Tags         Comments (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        status   query string false "Status filter"
// @Param        page     query int    false "Page number"    default(1)
// @Param        pageSize query int    false "Items per page" default(20)
// @Success      200 {object} object{comments=[]object,total=int}
// @Router       /admin/comments [get]
func (h *Handler) AdminList(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	p := handlerutil.ParsePagination(c, 20, handlerutil.DefaultMaxPageSize)
	comments, total, err := h.repo.ListAll(c.Request.Context(), status, p.Page, p.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments, "total": total, "page": p.Page, "pageSize": p.PageSize})
}

// AdminUpdateStatus updates a comment's status.
// @Summary      Update comment status
// @Description  Approve, reject, or mark a comment as spam
// @Tags         Comments (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Comment ID"
// @Param        body body object true "Status update"
// @Success      200 {object} object{message=string}
// @Router       /admin/comments/{id}/status [patch]
func (h *Handler) AdminUpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var input struct {
		Status CommentStatus `json:"status" binding:"required"`
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

// AdminDelete deletes a comment.
// @Summary      Delete comment
// @Description  Delete a comment by ID
// @Tags         Comments (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Comment ID"
// @Success      200 {object} object{message=string}
// @Router       /admin/comments/{id} [delete]
func (h *Handler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// AdminPin pins or unpins a comment.
// @Summary      Pin comment
// @Description  Set or unset the pinned flag on a comment
// @Tags         Comments (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Comment ID"
// @Param        body body object true "Pin state"
// @Success      200 {object} object{message=string}
// @Router       /admin/comments/{id}/pin [put]
func (h *Handler) AdminPin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
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

type adminReplyInput struct {
	Content     string `json:"content" binding:"required"`
	ParentID    *uint  `json:"parentId"`
	ContentType string `json:"contentType"`
	ContentID   uint   `json:"contentId"`
}

// AdminReply creates an approved author comment or reply (admin JWT required).
// @Summary      Reply as site author (admin)
// @Tags         Comments (Admin)
// @Security     BearerAuth
// @Router       /admin/comments/reply [post]
func (h *Handler) AdminReply(c *gin.Context) {
	var input adminReplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authorName := resolveSiteAuthorName(c.Request.Context(), h.siteCfgRepo, h.contentDocRepo)

	var contentType string
	var contentID uint
	var parentID *uint

	if input.ParentID != nil && *input.ParentID > 0 {
		parent, err := h.repo.FindByID(c.Request.Context(), *input.ParentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "parent comment not found"})
			return
		}
		contentType = parent.ContentType
		contentID = parent.ContentID
		parentID = &parent.ID
	} else {
		if input.ContentType == "" || input.ContentID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contentType and contentId required without parentId"})
			return
		}
		contentType = input.ContentType
		contentID = input.ContentID
	}

	comment := &Comment{
		Content:     input.Content,
		AuthorName:  authorName,
		ContentType: contentType,
		ContentID:   contentID,
		ParentID:    parentID,
		AuthorRole:  AuthorRoleAuthor,
		Status:      CommentStatusApproved,
	}
	if err := comment.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Create(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reply"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}
