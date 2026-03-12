package middleware

import (
	"github.com/gin-gonic/gin"
)

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
