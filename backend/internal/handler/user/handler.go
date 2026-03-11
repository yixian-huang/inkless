package user

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/auth"
)

// Handler handles user management HTTP requests
type Handler struct {
	userRepo repository.UserRepository
}

// NewHandler creates a new user handler
func NewHandler(userRepo repository.UserRepository) *Handler {
	return &Handler{userRepo: userRepo}
}

// UserResponse is the DTO for user responses (no password hash)
type UserResponse struct {
	ID           uint     `json:"id"`
	Username     string   `json:"username"`
	Role         string   `json:"role"`
	IsSuperAdmin bool     `json:"isSuperAdmin"`
	Permissions  []string `json:"permissions"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

func toUserResponse(u *model.User) UserResponse {
	perms := []string(u.Permissions)
	if perms == nil {
		perms = []string{}
	}
	return UserResponse{
		ID:           u.ID,
		Username:     u.Username,
		Role:         string(u.Role),
		IsSuperAdmin: u.IsSuperAdmin,
		Permissions:  perms,
		CreatedAt:    u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List returns a paginated list of users
// GET /admin/users
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	users, total, err := h.userRepo.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询用户失败"}})
		return
	}

	items := make([]UserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, toUserResponse(u))
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  page,
	})
}

// GetByID returns a single user
// GET /admin/users/:id
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "用户不存在"}})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// CreateRequest is the request body for creating a user
type CreateRequest struct {
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required,min=6"`
	Role        string   `json:"role" binding:"required"`
	Permissions []string `json:"permissions"`
}

// Create creates a new user
// POST /admin/users
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "请求参数错误: " + err.Error()}})
		return
	}

	// Validate role
	role := model.Role(req.Role)
	if !role.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "角色必须是 admin 或 editor"}})
		return
	}

	// Validate permissions
	if err := validatePermissions(req.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Check username uniqueness
	existing, _ := h.userRepo.FindByUsername(c.Request.Context(), req.Username)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "用户名已存在"}})
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "密码加密失败"}})
		return
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Role:         role,
		IsSuperAdmin: false, // Cannot set super admin via API
		Permissions:  req.Permissions,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "创建用户失败"}})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// UpdateRequest is the request body for updating a user
type UpdateRequest struct {
	Username    *string  `json:"username"`
	Password    *string  `json:"password"`
	Role        *string  `json:"role"`
	Permissions []string `json:"permissions"`
}

// Update updates a user
// PUT /admin/users/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "请求参数错误: " + err.Error()}})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "用户不存在"}})
		return
	}

	// Cannot modify super admin's role via API
	if user.IsSuperAdmin && req.Role != nil && model.Role(*req.Role) != user.Role {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "不能修改超级管理员的角色"}})
		return
	}

	if req.Username != nil {
		// Check uniqueness
		existing, _ := h.userRepo.FindByUsername(c.Request.Context(), *req.Username)
		if existing != nil && existing.ID != user.ID {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "用户名已存在"}})
			return
		}
		user.Username = *req.Username
	}

	if req.Password != nil {
		if len(*req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "密码长度不能少于6位"}})
			return
		}
		hashedPassword, err := auth.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "密码加密失败"}})
			return
		}
		user.PasswordHash = hashedPassword
	}

	if req.Role != nil {
		role := model.Role(*req.Role)
		if !role.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "角色必须是 admin 或 editor"}})
			return
		}
		user.Role = role
	}

	// Update permissions (only for non-super-admin users)
	if !user.IsSuperAdmin {
		if err := validatePermissions(req.Permissions); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		user.Permissions = req.Permissions
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "更新用户失败"}})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// Delete deletes a user
// DELETE /admin/users/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	// Get current user from context
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "未认证"}})
		return
	}

	// Cannot delete yourself
	if userCtx.UserID == uint(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "不能删除自己的账号"}})
		return
	}

	// Check if target user exists and is super admin
	targetUser, err := h.userRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "用户不存在"}})
		return
	}

	// If deleting a super admin, ensure at least one super admin remains
	if targetUser.IsSuperAdmin {
		count, err := h.userRepo.CountSuperAdmins(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询超管数量失败"}})
			return
		}
		if count <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "不能删除最后一个超级管理员"}})
			return
		}
	}

	if err := h.userRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "删除用户失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户已删除"})
}

// validatePermissions checks that all permissions are valid
func validatePermissions(perms []string) error {
	if perms == nil {
		return nil
	}
	validSet := make(map[string]bool, len(model.ValidPermissions))
	for _, p := range model.ValidPermissions {
		validSet[p] = true
	}
	var invalid []string
	for _, p := range perms {
		if !validSet[p] {
			invalid = append(invalid, p)
		}
	}
	if len(invalid) > 0 {
		return &invalidPermsError{perms: invalid}
	}
	return nil
}

type invalidPermsError struct {
	perms []string
}

func (e *invalidPermsError) Error() string {
	return "无效的权限: " + strings.Join(e.perms, ", ")
}
