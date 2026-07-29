package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	"github.com/yixian-huang/inkless/backend/pkg/auth"
)

// ContextKey defines the type for context keys to avoid collisions
type ContextKey string

const (
	// UserContextKey is the context key for storing authenticated user info
	UserContextKey ContextKey = "user"
	// apiKeyScopesKey holds scopes when authenticated via personal API key.
	apiKeyScopesKey = "api_key_scopes"
	// apiKeyIDKey holds the API key id for audit (optional).
	apiKeyIDKey = "api_key_id"
)

// UserContext represents the authenticated user information stored in context
type UserContext struct {
	UserID   uint
	Username string
	Role     model.Role
}

// APIKeyAuthenticator resolves long-lived API keys (optional second auth path).
type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, plaintext string) (*service.APIKeyPrincipal, error)
}

// Auth returns a middleware that validates bearer tokens (JWT or API key) and injects user context.
// apiKeys is optional; when nil, only JWT is accepted.
func Auth(jwtSecret string, apiKeys ...APIKeyAuthenticator) gin.HandlerFunc {
	var keyAuth APIKeyAuthenticator
	if len(apiKeys) > 0 {
		keyAuth = apiKeys[0]
	}
	return func(c *gin.Context) {
		// Extract bearer token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondWithError(c, apierror.Unauthorized("Missing authorization header"))
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondWithError(c, apierror.Unauthorized("Invalid authorization header format"))
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			respondWithError(c, apierror.Unauthorized("Missing bearer token"))
			return
		}

		// Long-lived API keys (ink_…) for PicGo / CLI
		if keyAuth != nil && strings.HasPrefix(tokenString, service.APIKeyPrefix) {
			p, err := keyAuth.Authenticate(c.Request.Context(), tokenString)
			if err != nil || p == nil {
				respondWithError(c, apierror.Unauthorized("Invalid API key"))
				return
			}
			if !p.Role.IsValid() {
				respondWithError(c, apierror.Unauthorized("Invalid role for API key owner"))
				return
			}
			c.Set(string(UserContextKey), &UserContext{
				UserID:   p.UserID,
				Username: p.Username,
				Role:     p.Role,
			})
			c.Set(apiKeyScopesKey, p.Scopes)
			c.Set(apiKeyIDKey, p.KeyID)
			c.Next()
			return
		}

		// Parse and validate JWT
		claims, err := auth.ParseToken(tokenString, jwtSecret)
		if err != nil {
			// Map auth errors to API errors
			if errors.Is(err, auth.ErrExpiredToken) {
				respondWithError(c, apierror.Unauthorized("Token has expired"))
				return
			}
			if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrInvalidClaims) {
				respondWithError(c, apierror.Unauthorized("Invalid token"))
				return
			}
			respondWithError(c, apierror.Unauthorized("Authentication failed"))
			return
		}

		// Validate role
		role := model.Role(claims.Role)
		if !role.IsValid() {
			respondWithError(c, apierror.Unauthorized("Invalid role in token"))
			return
		}

		// Inject user context
		userCtx := &UserContext{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     role,
		}
		c.Set(string(UserContextKey), userCtx)

		c.Next()
	}
}

// GetAPIKeyScopes returns scopes when the request used an API key; nil for JWT sessions.
func GetAPIKeyScopes(c *gin.Context) []string {
	if c == nil {
		return nil
	}
	v, ok := c.Get(apiKeyScopesKey)
	if !ok {
		return nil
	}
	scopes, ok := v.([]string)
	if !ok {
		return nil
	}
	return scopes
}

// GetAPIKeyID returns the personal API key id when the request used an ink_… key; 0 otherwise.
func GetAPIKeyID(c *gin.Context) uint {
	if c == nil {
		return 0
	}
	v, ok := c.Get(apiKeyIDKey)
	if !ok {
		return 0
	}
	switch id := v.(type) {
	case uint:
		return id
	case int:
		if id > 0 {
			return uint(id)
		}
	case int64:
		if id > 0 {
			return uint(id)
		}
	case uint64:
		return uint(id)
	}
	return 0
}

// RequireSessionJWT rejects personal API keys. Use for sensitive self-service
// routes such as creating/revoking API keys (must use a browser/session JWT).
func RequireSessionJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get(apiKeyIDKey); ok {
			respondWithError(c, apierror.Forbidden("Use a session JWT to manage API keys"))
			return
		}
		c.Next()
	}
}

// RequireAdmin returns a middleware that requires admin role
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		if userCtx.Role != model.RoleAdmin {
			respondWithError(c, apierror.Forbidden("Admin access required"))
			return
		}

		c.Next()
	}
}

// RequireAdminOrEditor returns a middleware that requires admin or editor role
func RequireAdminOrEditor() gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		if userCtx.Role != model.RoleAdmin && userCtx.Role != model.RoleEditor {
			respondWithError(c, apierror.Forbidden("Admin or editor access required"))
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin returns a middleware that requires super admin status
func RequireSuperAdmin(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		// Look up full user from database to check IsSuperAdmin
		user, err := userRepo.FindByID(c.Request.Context(), userCtx.UserID)
		if err != nil {
			respondWithError(c, apierror.Forbidden("User not found"))
			return
		}

		if !user.IsSuperAdmin {
			respondWithError(c, apierror.Forbidden("Super admin access required"))
			return
		}

		c.Next()
	}
}

// GetUserContext retrieves the authenticated user context from Gin context
func GetUserContext(c *gin.Context) *UserContext {
	val, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil
	}
	userCtx, ok := val.(*UserContext)
	if !ok {
		return nil
	}
	return userCtx
}

// respondWithError writes an APIError to the response and aborts the request
func respondWithError(c *gin.Context, err *apierror.APIError) {
	c.Set(auditFailureReasonKey, err.ErrorResponse.Message)
	c.JSON(err.HTTPStatus, gin.H{"error": err.ErrorResponse})
	c.Abort()
}
