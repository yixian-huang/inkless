package auth

import (
	"net/http"
	"time"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/pkg/apierror"
	"blotting-consultancy/pkg/auth"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	Role         string `json:"role"`
}

// Login authenticates a user and returns JWT tokens.
// @Summary      User login
// @Description  Authenticate with username and password, receive JWT access + refresh tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200 {object} LoginResponse
// @Failure      400 {object} object{error=string}
// @Failure      401 {object} object{error=string}
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.BadRequest("Invalid request body"))
		return
	}

	// Find user by username
	user, err := h.userRepo.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.Unauthorized("Invalid username or password"))
		return
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, apierror.Unauthorized("Invalid username or password"))
		return
	}

	// Generate token pair
	tokenPair, err := auth.GenerateTokenPair(
		user.ID,
		user.Username,
		string(user.Role),
		h.config.JWTSecret,
		h.config.JWTRefreshSecret,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.InternalServerError("Failed to generate tokens"))
		return
	}

	// Persist refresh token
	refreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(auth.RefreshTokenExpiry),
	}
	if err := h.refreshTokenRepo.Create(c.Request.Context(), refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, apierror.InternalServerError("Failed to persist refresh token"))
		return
	}

	// Return success response
	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int(auth.AccessTokenExpiry.Seconds()),
		Role:         string(user.Role),
	})
}
