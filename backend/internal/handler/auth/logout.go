package auth

import (
	"net/http"

	"blotting-consultancy/pkg/apierror"

	"github.com/gin-gonic/gin"
)

// LogoutRequest represents the logout request payload
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// LogoutResponse represents the logout response payload
type LogoutResponse struct {
	OK bool `json:"ok"`
}

// Logout invalidates the user's refresh token.
// @Summary      User logout
// @Description  Invalidate the refresh token to log out
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body LogoutRequest true "Refresh token to invalidate"
// @Success      200 {object} LogoutResponse
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.BadRequest("Invalid request body"))
		return
	}

	// Delete the refresh token from database (revoke it)
	err := h.refreshTokenRepo.DeleteByToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		// Even if token doesn't exist, we consider logout successful
		// This is idempotent behavior
		c.JSON(http.StatusOK, LogoutResponse{OK: true})
		return
	}

	c.JSON(http.StatusOK, LogoutResponse{OK: true})
}
