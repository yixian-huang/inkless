package scheduler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

type Handler struct {
	schedulerSvc *service.SchedulerService
}

func NewHandler(schedulerSvc *service.SchedulerService) *Handler {
	return &Handler{schedulerSvc: schedulerSvc}
}

type scheduleInput struct {
	ResourceType    model.ScheduledContentType `json:"resourceType" binding:"required"`
	ResourceID      uint                       `json:"resourceId" binding:"required"`
	ScheduledAt     time.Time                  `json:"scheduledAt" binding:"required"`
	ExpectedVersion *int                       `json:"expectedVersion"`
	PublishPayload  model.JSONMap              `json:"publishPayload"`
}

type rescheduleInput struct {
	ScheduledAt     time.Time     `json:"scheduledAt" binding:"required"`
	ExpectedVersion *int          `json:"expectedVersion"`
	PublishPayload  model.JSONMap `json:"publishPayload"`
}

type scheduledPublicationResponse struct {
	ID              uint                       `json:"id"`
	ResourceType    model.ScheduledContentType `json:"resourceType"`
	ResourceID      uint                       `json:"resourceId"`
	Title           string                     `json:"title"`
	Slug            string                     `json:"slug"`
	ScheduledAt     time.Time                  `json:"scheduledAt"`
	Status          model.ScheduledJobStatus   `json:"status"`
	Attempts        int                        `json:"attempts"`
	MaxAttempts     int                        `json:"maxAttempts"`
	ExpectedVersion *int                       `json:"expectedVersion,omitempty"`
	LastError       string                     `json:"lastError,omitempty"`
	LastAttemptAt   *time.Time                 `json:"lastAttemptAt,omitempty"`
	CompletedAt     *time.Time                 `json:"completedAt,omitempty"`
	CreatedAt       time.Time                  `json:"createdAt"`
	UpdatedAt       time.Time                  `json:"updatedAt"`
}

func userID(c *gin.Context) uint {
	uc := middleware.GetUserContext(c)
	if uc == nil {
		return 0
	}
	return uc.UserID
}

func parseJobID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid schedule id")
		return 0, false
	}
	return uint(id), true
}

func (h *Handler) List(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 50, 100)

	status := model.ScheduledJobStatus(c.Query("status"))
	if status != "" && !validStatus(status) {
		apierror.Write(c, apierror.BadRequest("invalid schedule status"))
		return
	}
	resourceID, ok := handlerutil.ParseUintParamOptional(c, "resourceId")
	if !ok {
		return
	}
	contentTypes, ok := allowedContentTypes(c, model.ScheduledContentType(c.Query("resourceType")))
	if !ok {
		return
	}

	items, total, err := h.schedulerSvc.List(
		c.Request.Context(),
		contentTypes,
		resourceID,
		status,
		p.Offset,
		p.PageSize,
	)
	if err != nil {
		apierror.Write(c, apierror.InternalServerError("failed to list schedule queue"))
		return
	}
	responses := make([]scheduledPublicationResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, h.response(c, item))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    responses,
		"total":    total,
		"page":     p.Page,
		"pageSize": p.PageSize,
	})
}

func (h *Handler) Schedule(c *gin.Context) {
	var input scheduleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}
	if !authorizeContentType(c, input.ResourceType) {
		return
	}
	middleware.SetAuditResource(c, contentResource(input.ResourceType, input.ResourceID))
	if !input.ScheduledAt.After(time.Now()) {
		apierror.Message(c, http.StatusBadRequest, "scheduledAt must be in the future")
		return
	}
	job, err := h.schedulerSvc.Schedule(
		c.Request.Context(),
		input.ResourceType,
		input.ResourceID,
		input.ScheduledAt,
		input.ExpectedVersion,
		input.PublishPayload,
		userID(c),
	)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.response(c, job))
}

func (h *Handler) Reschedule(c *gin.Context) {
	id, ok := parseJobID(c)
	if !ok {
		return
	}
	job, ok := h.authorizedJob(c, id)
	if !ok {
		return
	}
	if job.Status != model.ScheduledJobPending {
		apierror.Message(c, http.StatusConflict, "only pending schedules can be changed")
		return
	}
	middleware.SetAuditResource(c, contentResource(job.ContentType, job.ContentID))
	var input rescheduleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}
	if !input.ScheduledAt.After(time.Now()) {
		apierror.Message(c, http.StatusBadRequest, "scheduledAt must be in the future")
		return
	}
	updated, err := h.schedulerSvc.Reschedule(
		c.Request.Context(),
		id,
		input.ScheduledAt,
		input.ExpectedVersion,
		input.PublishPayload,
		userID(c),
	)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.response(c, updated))
}

func (h *Handler) Cancel(c *gin.Context) {
	id, ok := parseJobID(c)
	if !ok {
		return
	}
	job, ok := h.authorizedJob(c, id)
	if !ok {
		return
	}
	if job.Status != model.ScheduledJobPending {
		apierror.Message(c, http.StatusConflict, "only pending schedules can be cancelled")
		return
	}
	middleware.SetAuditResource(c, contentResource(job.ContentType, job.ContentID))
	cancelled, err := h.schedulerSvc.Cancel(c.Request.Context(), id, userID(c), time.Now())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.response(c, cancelled))
}

func (h *Handler) Retry(c *gin.Context) {
	id, ok := parseJobID(c)
	if !ok {
		return
	}
	job, ok := h.authorizedJob(c, id)
	if !ok {
		return
	}
	if job.Status != model.ScheduledJobFailed {
		apierror.Message(c, http.StatusConflict, "only failed schedules can be retried")
		return
	}
	middleware.SetAuditResource(c, contentResource(job.ContentType, job.ContentID))
	retried, err := h.schedulerSvc.Retry(c.Request.Context(), id, userID(c), time.Now())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.response(c, retried))
}

func (h *Handler) authorizedJob(c *gin.Context, id uint) (*model.ScheduledPublishJob, bool) {
	job, err := h.schedulerSvc.Get(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return nil, false
	}
	if !authorizeContentType(c, job.ContentType) {
		return nil, false
	}
	return job, true
}

func (h *Handler) response(c *gin.Context, job *model.ScheduledPublishJob) scheduledPublicationResponse {
	title, slug := h.schedulerSvc.DescribeResource(c.Request.Context(), job)
	var completedAt *time.Time
	switch job.Status {
	case model.ScheduledJobSucceeded:
		completedAt = job.SucceededAt
	case model.ScheduledJobFailed:
		completedAt = job.FailedAt
	case model.ScheduledJobCancelled:
		completedAt = job.CancelledAt
	}
	return scheduledPublicationResponse{
		ID:              job.ID,
		ResourceType:    job.ContentType,
		ResourceID:      job.ContentID,
		Title:           title,
		Slug:            slug,
		ScheduledAt:     job.ScheduledAt,
		Status:          job.Status,
		Attempts:        job.Attempts,
		MaxAttempts:     job.MaxAttempts,
		ExpectedVersion: job.ExpectedVersion,
		LastError:       job.LastError,
		LastAttemptAt:   job.LastAttemptAt,
		CompletedAt:     completedAt,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
	}
}

func allowedContentTypes(
	c *gin.Context,
	requested model.ScheduledContentType,
) ([]model.ScheduledContentType, bool) {
	if requested != "" {
		if !authorizeContentType(c, requested) {
			return nil, false
		}
		return []model.ScheduledContentType{requested}, true
	}
	user, ok := c.Get("rbac_user")
	if !ok {
		apierror.Message(c, http.StatusForbidden, "permission context is missing")
		return nil, false
	}
	rbacUser, ok := user.(*model.User)
	if !ok {
		apierror.Message(c, http.StatusForbidden, "permission context is invalid")
		return nil, false
	}
	types := make([]model.ScheduledContentType, 0, 2)
	if rbacUser.HasRBACPermission("articles", "publish") {
		types = append(types, model.ScheduledContentArticle)
	}
	if rbacUser.HasRBACPermission("pages", "publish") {
		types = append(types, model.ScheduledContentPage)
	}
	if len(types) == 0 {
		apierror.Message(c, http.StatusForbidden, "permission denied: publish permission required")
		return nil, false
	}
	return types, true
}

func authorizeContentType(c *gin.Context, contentType model.ScheduledContentType) bool {
	var resource string
	switch contentType {
	case model.ScheduledContentArticle:
		resource = "articles"
	case model.ScheduledContentPage:
		resource = "pages"
	default:
		apierror.Message(c, http.StatusBadRequest, "resourceType must be article or page")
		return false
	}
	user, ok := c.Get("rbac_user")
	rbacUser, valid := user.(*model.User)
	if !ok || !valid || !rbacUser.HasRBACPermission(resource, "publish") {
		apierror.Message(c, http.StatusForbidden, "permission denied: "+resource+":publish")
		return false
	}
	return true
}

func validStatus(status model.ScheduledJobStatus) bool {
	switch status {
	case model.ScheduledJobPending,
		model.ScheduledJobRunning,
		model.ScheduledJobSucceeded,
		model.ScheduledJobFailed,
		model.ScheduledJobCancelled:
		return true
	default:
		return false
	}
}

func contentResource(contentType model.ScheduledContentType, contentID uint) string {
	return string(contentType) + "s:" + strconv.FormatUint(uint64(contentID), 10)
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		apierror.Write(c, apierror.NotFound("schedule not found"))
	case errors.Is(err, service.ErrPageVersionConflict),
		errors.Is(err, service.ErrArticleVersionConflict):
		apierror.Write(c, apierror.Conflict(err.Error()))
	case strings.Contains(strings.ToLower(err.Error()), "unique"):
		apierror.Write(c, apierror.Conflict("another schedule is already active for this resource"))
	default:
		apierror.Write(c, apierror.BadRequest(err.Error()))
	}
}
