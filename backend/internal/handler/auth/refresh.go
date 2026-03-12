package auth

import (
	"net/http"

	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/auth"

	"github.com/gin-gonic/gin"
)

// RefreshRequest represents the refresh token request payload
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshResponse represents the refresh token response payload
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
}

// Refresh issues a new access token using a refresh token.
// @Summary      Refresh token
// @Description  Exchange a valid refresh token for a new access token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body RefreshRequest true "Refresh token"
// @Success      200 {object} RefreshResponse
// @Failure      401 {object} object{error=string}
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.BadRequest("Invalid request body"))
		return
	}

	// Parse and validate refresh token
	claims, err := auth.ParseToken(req.RefreshToken, h.config.JWTRefreshSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.Unauthorized("Invalid or expired refresh token"))
		return
	}

	// Verify refresh token exists in database
	storedToken, err := h.refreshTokenRepo.FindByToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.Unauthorized("Invalid or revoked refresh token"))
		return
	}

	// Check if token is expired
	if storedToken.IsExpired() {
		c.JSON(http.StatusUnauthorized, apierror.Unauthorized("Refresh token has expired"))
		return
	}

	// Generate new access token
	accessToken, err := auth.GenerateAccessToken(
		claims.UserID,
		claims.Username,
		claims.Role,
		h.config.JWTSecret,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.InternalServerError("Failed to generate access token"))
		return
	}

	// Return new access token
	c.JSON(http.StatusOK, RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(auth.AccessTokenExpiry.Seconds()),
	})
}
