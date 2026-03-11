package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/auth"
)

// ContextKey defines the type for context keys to avoid collisions
type ContextKey string

const (
	// UserContextKey is the context key for storing authenticated user info
	UserContextKey ContextKey = "user"
)

// UserContext represents the authenticated user information stored in context
type UserContext struct {
	UserID   uint
	Username string
	Role     model.Role
}

// Auth returns a middleware that validates bearer tokens and injects user context
func Auth(jwtSecret string) gin.HandlerFunc {
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

		// Parse and validate token
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
	c.JSON(err.HTTPStatus, gin.H{"error": err.ErrorResponse})
	c.Abort()
}
