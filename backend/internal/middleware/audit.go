package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/pkg/audit"
)

const auditActorHintKey = "audit_actor_hint"
const auditServiceHandledKey = "audit_service_handled"
const auditFailureReasonKey = "audit_failure_reason"
const maxLoginAuditBodyBytes = 64 << 10

// AuditContext is a Gin middleware that captures the client IP and User-Agent
// from the request and sets them in the Gin context for use by audit logging.
func AuditContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("client_ip", c.ClientIP())
		c.Set("user_agent", c.GetHeader("User-Agent"))
		c.Next()
	}
}

// GetAuditIP retrieves the client IP stored by AuditContext middleware.
func GetAuditIP(c *gin.Context) string {
	if val, exists := c.Get("client_ip"); exists {
		if ip, ok := val.(string); ok {
			return ip
		}
	}
	return c.ClientIP()
}

// GetAuditUserAgent retrieves the User-Agent stored by AuditContext middleware.
func GetAuditUserAgent(c *gin.Context) string {
	if val, exists := c.Get("user_agent"); exists {
		if ua, ok := val.(string); ok {
			return ua
		}
	}
	return c.GetHeader("User-Agent")
}

// AuditMutations records authenticated admin mutations. Unified-page publish,
// unpublish, and rollback successes are written by the service; this middleware
// still records authorization and handler failures for those operations.
func AuditMutations(writer audit.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		metadata := auditMetadata(c)
		c.Request = c.Request.WithContext(audit.WithMetadata(c.Request.Context(), metadata))
		c.Next()

		action, resource, serviceAudited := describeMutation(c.Request.Method, c.FullPath(), c)
		if action == "" || writer == nil {
			return
		}
		status := c.Writer.Status()
		if serviceAudited {
			if handled, ok := c.Get(auditServiceHandledKey); ok && handled == true {
				return
			}
		}

		event := audit.Event{
			Action:   action,
			Actor:    metadata.ActorLabel(),
			Resource: resource,
			Result:   auditResult(status),
			Details: audit.AddMetadata(map[string]interface{}{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
				"route":  c.FullPath(),
				"status": status,
			}, metadata),
		}
		if status >= http.StatusBadRequest {
			event.Details["reason"] = failureReason(c, status)
		}
		_ = writer.Write(context.WithoutCancel(c.Request.Context()), event)
	}
}

// MarkAuditHandled tells AuditMutations that the called service owns the audit
// event for this operation.
func MarkAuditHandled(c *gin.Context) {
	c.Set(auditServiceHandledKey, true)
}

// AuditLogin records successful and failed login attempts without persisting
// credentials. The request body is restored before the login handler runs.
func AuditLogin(writer audit.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := readLoginUsername(c)
		if username == "" {
			username = "anonymous"
		}
		c.Set(auditActorHintKey, username)
		metadata := auditMetadata(c)
		c.Request = c.Request.WithContext(audit.WithMetadata(c.Request.Context(), metadata))
		c.Next()

		if writer == nil {
			return
		}
		status := c.Writer.Status()
		event := audit.Event{
			Action:   "auth.login",
			Actor:    metadata.ActorLabel(),
			Resource: "auth:login",
			Result:   auditResult(status),
			Details: audit.AddMetadata(map[string]interface{}{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
				"route":  c.FullPath(),
				"status": status,
			}, metadata),
		}
		if status >= http.StatusBadRequest {
			event.Details["reason"] = failureReason(c, status)
		}
		_ = writer.Write(context.WithoutCancel(c.Request.Context()), event)
	}
}

func auditMetadata(c *gin.Context) audit.Metadata {
	metadata := audit.Metadata{
		IP:        GetAuditIP(c),
		UserAgent: GetAuditUserAgent(c),
		RequestID: firstNonEmpty(c.GetHeader("X-Request-ID"), c.GetHeader("X-Correlation-ID")),
	}
	if user := GetUserContext(c); user != nil {
		metadata.Actor = user.Username
		metadata.ActorID = user.UserID
	}
	if metadata.Actor == "" {
		if actor, ok := c.Get(auditActorHintKey); ok {
			metadata.Actor, _ = actor.(string)
		}
	}
	return metadata
}

func describeMutation(method, route string, c *gin.Context) (action, resource string, serviceAudited bool) {
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete {
		return "", "", false
	}

	resource = auditResource(route, c)
	switch route {
	case "/admin/pages":
		if method == http.MethodPost {
			return "content.create", resource, false
		}
	case "/admin/pages/:id":
		switch method {
		case http.MethodPut, http.MethodPatch:
			return "content.update", resource, false
		case http.MethodDelete:
			return "content.delete", resource, false
		}
	case "/admin/pages/:id/draft":
		return "content.save_draft", resource, false
	case "/admin/pages/:id/publish":
		return "content.publish", resource, true
	case "/admin/pages/:id/unpublish":
		return "content.unpublish", resource, true
	case "/admin/pages/:id/rollback":
		return "content.rollback", resource, true
	case "/admin/articles":
		if method == http.MethodPost {
			return "content.create", resource, false
		}
	case "/admin/articles/:id":
		if method == http.MethodDelete {
			return "content.delete", resource, false
		}
		return "content.update", resource, false
	case "/admin/roles/assign":
		return "permissions.assign", resource, false
	case "/admin/roles/unassign":
		return "permissions.unassign", resource, false
	case "/admin/migration/import":
		return "migration.import", resource, false
	case "/admin/backups/trigger":
		return "backup.create", resource, false
	case "/admin/backups/import":
		return "backup.restore", resource, false
	case "/admin/backups/export":
		return "backup.export", resource, false
	}

	name := firstRouteSegment(route)
	if name == "" {
		return "", "", false
	}
	operation := mutationOperation(method, route)
	action = name + "." + operation
	if len(action) > 50 {
		action = action[:50]
	}
	return action, resource, false
}

func auditResource(route string, c *gin.Context) string {
	name := firstRouteSegment(route)
	if name == "" {
		name = "admin"
	}
	for _, key := range []string{"id", "itemId", "jobId", "userId", "slug", "filename"} {
		if value := c.Param(key); value != "" {
			resource := name + ":" + value
			if len(resource) > 100 {
				return resource[:100]
			}
			return resource
		}
	}
	return name
}

func firstRouteSegment(route string) string {
	route = strings.TrimPrefix(route, "/admin/")
	route = strings.TrimPrefix(route, "/auth/")
	route = strings.Trim(route, "/")
	if route == "" {
		return ""
	}
	segment, _, _ := strings.Cut(route, "/")
	return strings.ReplaceAll(segment, "_", "-")
}

func mutationOperation(method, route string) string {
	last := route
	if index := strings.LastIndex(route, "/"); index >= 0 {
		last = route[index+1:]
	}
	if last != "" && !strings.HasPrefix(last, ":") {
		switch last {
		case "activate", "apply", "assign", "complete", "export", "import", "install",
			"move", "primary", "publish", "reorder", "restore", "test", "trigger",
			"unassign", "uninstall", "unpublish", "update", "validate":
			return strings.ReplaceAll(last, "_", "-")
		}
	}
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodDelete:
		return "delete"
	default:
		return "update"
	}
}

func readLoginUsername(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}
	originalBody := c.Request.Body
	body, err := io.ReadAll(io.LimitReader(originalBody, maxLoginAuditBodyBytes+1))
	c.Request.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.MultiReader(bytes.NewReader(body), originalBody),
		Closer: originalBody,
	}
	if err != nil {
		return ""
	}
	if len(body) > maxLoginAuditBodyBytes {
		return ""
	}

	var payload struct {
		Username string `json:"username"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.Username)
}

func auditResult(status int) string {
	if status >= http.StatusBadRequest {
		return "failure"
	}
	return "success"
}

func failureReason(c *gin.Context, status int) string {
	if reason, ok := c.Get(auditFailureReasonKey); ok {
		if text, valid := reason.(string); valid && text != "" {
			return text
		}
	}
	if len(c.Errors) > 0 {
		return c.Errors.Last().Error()
	}
	if reason := http.StatusText(status); reason != "" {
		return reason
	}
	return "request failed"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
