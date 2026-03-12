package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/apierror"
)

// RequirePermission returns middleware that checks if the authenticated user
// has a specific permission (resource:action) via their assigned RBAC roles.
// Falls back to legacy Role field check for backward compatibility.
func RequirePermission(resource, action string, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		// Load user with roles and permissions
		user, err := userRepo.FindByIDWithRoles(c.Request.Context(), userCtx.UserID)
		if err != nil {
			respondWithError(c, apierror.Forbidden("User not found"))
			return
		}

		if !user.HasRBACPermission(resource, action) {
			respondWithError(c, apierror.Forbidden(
				fmt.Sprintf("Permission denied: %s:%s", resource, action),
			))
			return
		}

		// Inject full user into context for downstream handlers
		c.Set("rbac_user", user)
		c.Next()
	}
}

// RequireAnyPermission returns middleware that checks if the user has at least
// one of the given permissions.
func RequireAnyPermission(perms []PermissionPair, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		// Load user with roles and permissions
		user, err := userRepo.FindByIDWithRoles(c.Request.Context(), userCtx.UserID)
		if err != nil {
			respondWithError(c, apierror.Forbidden("User not found"))
			return
		}

		for _, p := range perms {
			if user.HasRBACPermission(p.Resource, p.Action) {
				c.Set("rbac_user", user)
				c.Next()
				return
			}
		}

		respondWithError(c, apierror.Forbidden("Permission denied: insufficient permissions"))
	}
}

// PermissionPair represents a resource:action permission pair
type PermissionPair struct {
	Resource string
	Action   string
}
