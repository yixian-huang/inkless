package auth

import (
	"net/http"

	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
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

// Me handles GET /auth/me
// This endpoint requires authentication middleware to be applied
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

	// Fetch full user from database to get permissions
	fullUser, err := h.userRepo.FindByID(c.Request.Context(), user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.InternalServerError("Failed to fetch user info"))
		return
	}

	// Build permissions list
	permissions := fullUser.Permissions
	if fullUser.IsSuperAdmin {
		permissions = model.ValidPermissions
	}
	if permissions == nil {
		permissions = []string{}
	}

	c.JSON(http.StatusOK, MeResponse{
		ID:           fullUser.ID,
		Username:     fullUser.Username,
		Role:         string(fullUser.Role),
		IsSuperAdmin: fullUser.IsSuperAdmin,
		Permissions:  permissions,
	})
}
