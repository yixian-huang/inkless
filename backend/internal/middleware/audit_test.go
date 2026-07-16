package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/pkg/audit"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditWriterStub struct {
	events []audit.Event
	err    error
}

func (w *auditWriterStub) Write(_ context.Context, event audit.Event) error {
	w.events = append(w.events, event)
	return w.err
}

func TestAuditContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedIP, capturedUA string

	router := gin.New()
	router.Use(AuditContext())
	router.GET("/test", func(c *gin.Context) {
		capturedIP = GetAuditIP(c)
		capturedUA = GetAuditUserAgent(c)
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.RemoteAddr = "192.168.1.100:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "192.168.1.100", capturedIP)
	assert.Equal(t, "TestAgent/1.0", capturedUA)
}

func TestGetAuditIP_Fallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "10.0.0.1:9999"

	// Without AuditContext middleware, falls back to ClientIP()
	ip := GetAuditIP(c)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestGetAuditUserAgent_Fallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("User-Agent", "FallbackAgent")

	ua := GetAuditUserAgent(c)
	assert.Equal(t, "FallbackAgent", ua)
}

func TestAuditMutations_RecordsPermissionFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{}

	router := gin.New()
	router.Use(AuditContext())
	router.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), &UserContext{UserID: 7, Username: "editor", Role: "editor"})
		c.Next()
	})
	router.Use(AuditMutations(writer))
	router.POST("/admin/pages/:id/publish", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/pages/42/publish", nil)
	req.Header.Set("User-Agent", "AuditTest/1.0")
	req.Header.Set("X-Request-ID", "req-42")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Len(t, writer.events, 1)
	event := writer.events[0]
	assert.Equal(t, "content.publish", event.Action)
	assert.Equal(t, "editor", event.Actor)
	assert.Equal(t, "pages:42", event.Resource)
	assert.Equal(t, "failure", event.Result)
	assert.Equal(t, http.StatusForbidden, event.Details["status"])
	assert.Equal(t, "Forbidden", event.Details["reason"])
	assert.Equal(t, "req-42", event.Details["request_id"])
}

func TestAuditMutations_CapturesRBACFailureReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{}
	repo := &rbacTestUserRepository{
		user: &model.User{ID: 7, Role: model.RoleEditor},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), &UserContext{UserID: 7, Username: "editor", Role: model.RoleEditor})
		c.Next()
	})
	router.Use(AuditMutations(writer))
	router.POST(
		"/admin/pages/:id/publish",
		RequirePermission("pages", "publish", repo, nil),
		okHandler,
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/pages/42/publish", nil))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Len(t, writer.events, 1)
	assert.Equal(t, "Permission denied: pages:publish", writer.events[0].Details["reason"])
}

func TestAuditMutations_CapturesLegacyRoleBoundaryFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), &UserContext{UserID: 8, Username: "viewer", Role: model.Role("viewer")})
		c.Next()
	})
	router.Use(AuditMutations(writer))
	router.Use(RequireAdminOrEditor())
	router.POST("/admin/pages", okHandler)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/pages", nil))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Len(t, writer.events, 1)
	assert.Equal(t, "content.create", writer.events[0].Action)
	assert.Equal(t, "Admin or editor access required", writer.events[0].Details["reason"])
}

func TestDescribeMutation_UnifiedPageLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		method         string
		route          string
		wantAction     string
		serviceAudited bool
	}{
		{http.MethodPost, "/admin/pages", "content.create", false},
		{http.MethodPut, "/admin/pages/:id", "content.update", false},
		{http.MethodPut, "/admin/pages/:id/draft", "content.save_draft", false},
		{http.MethodPost, "/admin/pages/:id/publish", "content.publish", true},
		{http.MethodPost, "/admin/pages/:id/unpublish", "content.unpublish", true},
		{http.MethodPost, "/admin/pages/:id/rollback", "content.rollback", true},
		{http.MethodDelete, "/admin/pages/:id", "content.delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.wantAction, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Params = gin.Params{{Key: "id", Value: "42"}}
			action, resource, serviceAudited := describeMutation(tt.method, tt.route, c)
			assert.Equal(t, tt.wantAction, action)
			assert.Equal(t, "pages:42", resource)
			assert.Equal(t, tt.serviceAudited, serviceAudited)
		})
	}
}

func TestAuditMutations_ServiceOwnedSuccessIsNotDuplicated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{}

	router := gin.New()
	router.Use(AuditMutations(writer))
	router.POST("/admin/pages/:id/publish", func(c *gin.Context) {
		MarkAuditHandled(c)
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/pages/42/publish", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, writer.events)
}

func TestAuditMutations_WriteFailureDoesNotChangeBusinessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{err: errors.New("audit database unavailable")}

	router := gin.New()
	router.Use(AuditMutations(writer))
	router.PUT("/admin/settings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"saved": true})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/admin/settings", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, writer.events, 1)
	assert.Equal(t, "settings.update", writer.events[0].Action)
}

func TestAuditLogin_RestoresBodyAndRecordsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{}

	router := gin.New()
	router.Use(AuditContext())
	router.POST("/auth/login", AuditLogin(writer), func(c *gin.Context) {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		require.NoError(t, c.ShouldBindJSON(&input))
		assert.Equal(t, "alice", input.Username)
		assert.Equal(t, "secret", input.Password)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		bytes.NewBufferString(`{"username":"alice","password":"secret"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Len(t, writer.events, 1)
	assert.Equal(t, "auth.login", writer.events[0].Action)
	assert.Equal(t, "alice", writer.events[0].Actor)
	assert.Equal(t, "failure", writer.events[0].Result)
	assert.NotContains(t, writer.events[0].Details, "password")
}

func TestAuditLogin_LargeBodyIsNotBufferedAndRemainsReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := &auditWriterStub{}
	body := `{"username":"` + strings.Repeat("a", maxLoginAuditBodyBytes) + `","password":"secret"}`

	router := gin.New()
	router.POST("/auth/login", AuditLogin(writer), func(c *gin.Context) {
		restored, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(restored))
		c.Status(http.StatusBadRequest)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body)),
	)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Len(t, writer.events, 1)
	assert.Equal(t, "anonymous", writer.events[0].Actor)
}
