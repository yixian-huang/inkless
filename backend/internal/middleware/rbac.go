package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/apierror"
)

// loadUserWithCache loads a user with roles/permissions, using a short-lived cache
// to avoid hitting the DB on every admin request.
func loadUserWithCache(c *gin.Context, userID uint, userRepo repository.UserRepository, rbacCache *cache.Cache) (*model.User, error) {
	if userRepo == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	cacheKey := fmt.Sprintf("rbac:%d", userID)
	if rbacCache != nil {
		if cached, ok := rbacCache.Get(cacheKey); ok {
			user, valid := cached.(*model.User)
			if !valid {
				return nil, fmt.Errorf("invalid RBAC cache entry for user %d", userID)
			}
			return user, nil
		}
	}
	user, err := userRepo.FindByIDWithRoles(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	if rbacCache != nil {
		rbacCache.Set(cacheKey, user)
	}
	return user, nil
}

// RequirePermission returns middleware that checks if the authenticated user
// has a specific permission (resource:action) via their assigned RBAC roles.
// Falls back to legacy Role field check for backward compatibility.
func RequirePermission(resource, action string, userRepo repository.UserRepository, rbacCache *cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		user, err := loadUserWithCache(c, userCtx.UserID, userRepo, rbacCache)
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

		c.Set("rbac_user", user)
		c.Next()
	}
}

// RequireAnyPermission returns middleware that checks if the user has at least
// one of the given permissions.
func RequireAnyPermission(perms []PermissionPair, userRepo repository.UserRepository, rbacCache *cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			respondWithError(c, apierror.Unauthorized("Authentication required"))
			return
		}

		user, err := loadUserWithCache(c, userCtx.UserID, userRepo, rbacCache)
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
