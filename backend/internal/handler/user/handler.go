package user

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/pkg/auth"
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

// List returns a paginated list of users.
// @Summary      List users
// @Description  Returns paginated list of users
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int false "Page number"    default(1)
// @Param        pageSize query int false "Items per page" default(20)
// @Success      200 {object} object{items=[]object,total=int,page=int}
// @Router       /admin/users [get]
func (h *Handler) List(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 20, 100)
	page, pageSize := p.Page, p.PageSize
	offset := p.Offset

	users, total, err := h.userRepo.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询用户失败")
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

// GetByID returns a single user.
// @Summary      Get user by ID
// @Description  Returns a single user by ID
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "User ID"
// @Success      200 {object} UserResponse
// @Failure      404 {object} object{error=string}
// @Router       /admin/users/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "用户不存在")
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

// Create creates a new user.
// @Summary      Create user
// @Description  Create a new user with username, password, and role
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateRequest true "User data"
// @Success      201 {object} UserResponse
// @Failure      400 {object} object{error=string}
// @Failure      409 {object} object{error=string}
// @Router       /admin/users [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// Validate role
	role := model.Role(req.Role)
	if !role.IsValid() {
		apierror.Message(c, http.StatusBadRequest, "角色必须是 admin 或 editor")
		return
	}

	// Validate permissions
	if err := validatePermissions(req.Permissions); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check username uniqueness
	existing, _ := h.userRepo.FindByUsername(c.Request.Context(), req.Username)
	if existing != nil {
		apierror.Message(c, http.StatusConflict, "用户名已存在")
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "密码加密失败")
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
		apierror.Message(c, http.StatusInternalServerError, "创建用户失败")
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

// Update updates a user.
// @Summary      Update user
// @Description  Update user profile, password, role, or permissions
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int           true "User ID"
// @Param        body body UpdateRequest true "Updated user data"
// @Success      200 {object} UserResponse
// @Failure      404 {object} object{error=string}
// @Router       /admin/users/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "用户不存在")
		return
	}

	// Cannot modify super admin's role via API
	if user.IsSuperAdmin && req.Role != nil && model.Role(*req.Role) != user.Role {
		apierror.Message(c, http.StatusForbidden, "不能修改超级管理员的角色")
		return
	}

	if req.Username != nil {
		// Check uniqueness
		existing, _ := h.userRepo.FindByUsername(c.Request.Context(), *req.Username)
		if existing != nil && existing.ID != user.ID {
			apierror.Message(c, http.StatusConflict, "用户名已存在")
			return
		}
		user.Username = *req.Username
	}

	if req.Password != nil {
		if len(*req.Password) < 6 {
			apierror.Message(c, http.StatusBadRequest, "密码长度不能少于6位")
			return
		}
		hashedPassword, err := auth.HashPassword(*req.Password)
		if err != nil {
			apierror.Message(c, http.StatusInternalServerError, "密码加密失败")
			return
		}
		user.PasswordHash = hashedPassword
	}

	if req.Role != nil {
		role := model.Role(*req.Role)
		if !role.IsValid() {
			apierror.Message(c, http.StatusBadRequest, "角色必须是 admin 或 editor")
			return
		}
		user.Role = role
	}

	// Update permissions (only for non-super-admin users)
	if !user.IsSuperAdmin {
		if err := validatePermissions(req.Permissions); err != nil {
			apierror.Message(c, http.StatusBadRequest, err.Error())
			return
		}
		user.Permissions = req.Permissions
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "更新用户失败")
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// Delete deletes a user.
// @Summary      Delete user
// @Description  Delete a user by ID (cannot delete self or last super admin)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "User ID"
// @Success      200 {object} object{message=string}
// @Failure      400 {object} object{error=string}
// @Router       /admin/users/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Get current user from context
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		apierror.Message(c, http.StatusUnauthorized, "未认证")
		return
	}

	// Cannot delete yourself
	if userCtx.UserID == id {
		apierror.Message(c, http.StatusBadRequest, "不能删除自己的账号")
		return
	}

	// Check if target user exists and is super admin
	targetUser, err := h.userRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "用户不存在")
		return
	}

	// If deleting a super admin, ensure at least one super admin remains
	if targetUser.IsSuperAdmin {
		count, err := h.userRepo.CountSuperAdmins(c.Request.Context())
		if err != nil {
			apierror.Message(c, http.StatusInternalServerError, "查询超管数量失败")
			return
		}
		if count <= 1 {
			apierror.Message(c, http.StatusBadRequest, "不能删除最后一个超级管理员")
			return
		}
	}

	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "删除用户失败")
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
