package auditlog

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// Handler handles audit log HTTP requests
type Handler struct {
	auditEventRepo repository.AuditEventRepository
}

// NewHandler creates a new audit log handler
func NewHandler(auditEventRepo repository.AuditEventRepository) *Handler {
	return &Handler{auditEventRepo: auditEventRepo}
}

// List returns a paginated, filtered list of audit events.
// @Summary      List audit logs
// @Description  Returns paginated audit events with optional filters
// @Tags         Audit Logs
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int    false "Page number"    default(1)
// @Param        pageSize query int    false "Items per page" default(20)
// @Param        action   query string false "Action filter"
// @Param        actor    query string false "Actor filter"
// @Param        from     query string false "From date (RFC3339)"
// @Param        to       query string false "To date (RFC3339)"
// @Success      200 {object} object{items=[]object,total=int,page=int,pageSize=int}
// @Router       /admin/audit-logs [get]
func (h *Handler) List(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 20, 100)
	page, pageSize := p.Page, p.PageSize
	offset := p.Offset

	action := c.Query("action")
	actor := c.Query("actor")

	var from, to *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		t, err := parseAuditTime(fromStr, false)
		if err != nil {
			apierror.Message(c, http.StatusBadRequest, "from 必须是 RFC3339 时间或 YYYY-MM-DD 日期")
			return
		}
		from = &t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := parseAuditTime(toStr, true)
		if err != nil {
			apierror.Message(c, http.StatusBadRequest, "to 必须是 RFC3339 时间或 YYYY-MM-DD 日期")
			return
		}
		to = &t
	}

	items, total, err := h.auditEventRepo.List(c.Request.Context(), offset, pageSize, action, actor, from, to)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询审计日志失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func parseAuditTime(value string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return parsed.Add(24*time.Hour - time.Nanosecond), nil
	}
	return parsed, nil
}
