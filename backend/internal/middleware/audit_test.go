package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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
