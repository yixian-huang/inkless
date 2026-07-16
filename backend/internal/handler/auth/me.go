package auth

import (
	"net/http"

	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/pkg/apierror"

	"github.com/gin-gonic/gin"
)

// MeResponse represents the current user response payload
type MeResponse struct {
	ID           uint     `json:"id"`
	Username     string   `json:"username"`
	Role         string   `json:"role"`
	IsSuperAdmin bool     `json:"isSuperAdmin"`
	Permissions  []string `json:"permissions"`
}

// Me returns the current authenticated user's profile.
// @Summary      Get current user
// @Description  Returns profile of the currently authenticated user
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} MeResponse
// @Failure      401 {object} object{error=string}
// @Router       /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	// Extract user context from middleware
	userCtx, exists := c.Get(string(middleware.UserContextKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, apierror.Unauthorized("User context not found"))
		return
	}

	user, ok := userCtx.(*middleware.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, apierror.InternalServerError("Invalid user context"))
		return
	}

	// Fetch roles and permissions so the response reflects backend-enforced RBAC.
	fullUser, err := h.userRepo.FindByIDWithRoles(c.Request.Context(), user.UserID)
	if err != nil {
		// Older installations may not have RBAC tables yet. Keep /auth/me
		// available during migration and derive permissions from the legacy role.
		fullUser, err = h.userRepo.FindByID(c.Request.Context(), user.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, apierror.InternalServerError("Failed to fetch user info"))
			return
		}
	}

	permissions := fullUser.EffectivePermissionKeys()

	c.JSON(http.StatusOK, MeResponse{
		ID:           fullUser.ID,
		Username:     fullUser.Username,
		Role:         string(fullUser.Role),
		IsSuperAdmin: fullUser.IsSuperAdmin,
		Permissions:  permissions,
	})
}
