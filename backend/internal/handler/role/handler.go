package role

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles RBAC role management HTTP requests
type Handler struct {
	roleRepo repository.RoleRepository
	userRepo repository.UserRepository
}

// NewHandler creates a new role handler
func NewHandler(roleRepo repository.RoleRepository, userRepo repository.UserRepository) *Handler {
	return &Handler{roleRepo: roleRepo, userRepo: userRepo}
}

// RoleResponse is the DTO for role responses
type RoleResponse struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description"`
	IsSystem    bool                 `json:"isSystem"`
	Permissions []PermissionResponse `json:"permissions"`
	UserCount   int64                `json:"userCount"`
	CreatedAt   string               `json:"createdAt"`
	UpdatedAt   string               `json:"updatedAt"`
}

// PermissionResponse is the DTO for permission responses
type PermissionResponse struct {
	ID          uint   `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

func toRoleResponse(r *model.RBACRole, userCount int64) RoleResponse {
	perms := make([]PermissionResponse, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, PermissionResponse{
			ID:          p.ID,
			Resource:    p.Resource,
			Action:      p.Action,
			Key:         p.Key(),
			Description: p.Description,
		})
	}
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Permissions: perms,
		UserCount:   userCount,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List returns all roles.
// @Summary      List roles
// @Description  Returns all roles with their permissions
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} RoleResponse
// @Router       /admin/roles [get]
func (h *Handler) List(c *gin.Context) {
	roles, err := h.roleRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to list roles"}})
		return
	}

	items := make([]RoleResponse, 0, len(roles))
	for _, r := range roles {
		count, _ := h.roleRepo.CountUsersWithRole(c.Request.Context(), r.ID)
		items = append(items, toRoleResponse(r, count))
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetByID returns a single role with permissions.
// @Summary      Get role by ID
// @Description  Returns a single role with its permissions
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Role ID"
// @Success      200 {object} RoleResponse
// @Router       /admin/roles/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid ID"}})
		return
	}

	role, err := h.roleRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Role not found"}})
		return
	}

	count, _ := h.roleRepo.CountUsersWithRole(c.Request.Context(), role.ID)
	c.JSON(http.StatusOK, toRoleResponse(role, count))
}

// CreateRequest is the request body for creating a role
type CreateRequest struct {
	Name          string `json:"name" binding:"required"`
	DisplayName   string `json:"displayName" binding:"required"`
	Description   string `json:"description"`
	PermissionIDs []uint `json:"permissionIds"`
}

// Create creates a new custom role.
// @Summary      Create role
// @Description  Create a new custom role with permissions
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateRequest true "Role data"
// @Success      201 {object} RoleResponse
// @Router       /admin/roles [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid request: " + err.Error()}})
		return
	}

	role := &model.RBACRole{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsSystem:    false, // custom roles are never system roles
	}

	if err := role.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Check name uniqueness
	existing, _ := h.roleRepo.FindByName(c.Request.Context(), req.Name)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "Role name already exists"}})
		return
	}

	if err := h.roleRepo.Create(c.Request.Context(), role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to create role"}})
		return
	}

	// Assign permissions if provided
	if len(req.PermissionIDs) > 0 {
		if err := h.roleRepo.SetPermissions(c.Request.Context(), role.ID, req.PermissionIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to set permissions"}})
			return
		}
	}

	// Reload with permissions
	role, _ = h.roleRepo.FindByID(c.Request.Context(), role.ID)
	c.JSON(http.StatusCreated, toRoleResponse(role, 0))
}

// UpdateRequest is the request body for updating a role
type UpdateRequest struct {
	DisplayName   *string `json:"displayName"`
	Description   *string `json:"description"`
	PermissionIDs []uint  `json:"permissionIds"`
}

// Update updates a role.
// @Summary      Update role
// @Description  Update role details and permissions. System roles can only have permissions updated.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int           true "Role ID"
// @Param        body body UpdateRequest true "Updated role data"
// @Success      200 {object} RoleResponse
// @Router       /admin/roles/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid ID"}})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid request: " + err.Error()}})
		return
	}

	role, err := h.roleRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Role not found"}})
		return
	}

	// For non-system roles, allow updating display name and description
	if !role.IsSystem {
		if req.DisplayName != nil {
			role.DisplayName = *req.DisplayName
		}
		if req.Description != nil {
			role.Description = *req.Description
		}
		if err := h.roleRepo.Update(c.Request.Context(), role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to update role"}})
			return
		}
	}

	// Update permissions (allowed for all roles, including system roles)
	if req.PermissionIDs != nil {
		if err := h.roleRepo.SetPermissions(c.Request.Context(), role.ID, req.PermissionIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to update permissions"}})
			return
		}
	}

	// Reload with permissions
	role, _ = h.roleRepo.FindByID(c.Request.Context(), role.ID)
	count, _ := h.roleRepo.CountUsersWithRole(c.Request.Context(), role.ID)
	c.JSON(http.StatusOK, toRoleResponse(role, count))
}

// Delete deletes a role.
// @Summary      Delete role
// @Description  Delete a role (cannot delete system roles or roles with assigned users)
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Role ID"
// @Success      200 {object} object{message=string}
// @Router       /admin/roles/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid ID"}})
		return
	}

	role, err := h.roleRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Role not found"}})
		return
	}

	// Cannot delete system roles
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Cannot delete system roles"}})
		return
	}

	// Cannot delete roles with assigned users
	count, err := h.roleRepo.CountUsersWithRole(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to check role usage"}})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "Cannot delete role with assigned users"}})
		return
	}

	if err := h.roleRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to delete role"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}

// ListPermissions returns all available permissions.
// @Summary      List permissions
// @Description  Returns all available permissions
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} PermissionResponse
// @Router       /admin/permissions [get]
func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.roleRepo.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to list permissions"}})
		return
	}

	items := make([]PermissionResponse, 0, len(perms))
	for _, p := range perms {
		items = append(items, PermissionResponse{
			ID:          p.ID,
			Resource:    p.Resource,
			Action:      p.Action,
			Key:         p.Key(),
			Description: p.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// AssignRoleRequest is the request body for assigning a role to a user
type AssignRoleRequest struct {
	UserID uint  `json:"userId" binding:"required"`
	RoleID uint  `json:"roleId" binding:"required"`
	SiteID *uint `json:"siteId"`
}

// AssignRole assigns a role to a user.
// @Summary      Assign role to user
// @Description  Assign an RBAC role to a user
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body AssignRoleRequest true "Assignment data"
// @Success      200 {object} object{message=string}
// @Router       /admin/roles/assign [post]
func (h *Handler) AssignRole(c *gin.Context) {
	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid request: " + err.Error()}})
		return
	}

	// Verify user exists
	_, err := h.userRepo.FindByID(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "User not found"}})
		return
	}

	// Verify role exists
	_, err = h.roleRepo.FindByID(c.Request.Context(), req.RoleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Role not found"}})
		return
	}

	if err := h.roleRepo.AssignRoleToUser(c.Request.Context(), req.UserID, req.RoleID, req.SiteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to assign role: " + err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

// UnassignRoleRequest is the request body for removing a role from a user
type UnassignRoleRequest struct {
	UserID uint `json:"userId" binding:"required"`
	RoleID uint `json:"roleId" binding:"required"`
}

// UnassignRole removes a role from a user.
// @Summary      Remove role from user
// @Description  Remove an RBAC role from a user
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body UnassignRoleRequest true "Unassignment data"
// @Success      200 {object} object{message=string}
// @Router       /admin/roles/unassign [post]
func (h *Handler) UnassignRole(c *gin.Context) {
	var req UnassignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid request: " + err.Error()}})
		return
	}

	if err := h.roleRepo.RemoveRoleFromUser(c.Request.Context(), req.UserID, req.RoleID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Role assignment not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

// GetUserRoles returns all roles assigned to a user.
// @Summary      Get user roles
// @Description  Returns all RBAC roles assigned to a user
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        userId path int true "User ID"
// @Success      200 {object} object{items=[]RoleResponse}
// @Router       /admin/roles/user/{userId} [get]
func (h *Handler) GetUserRoles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid user ID"}})
		return
	}

	userRoles, err := h.roleRepo.GetUserRoles(c.Request.Context(), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to get user roles"}})
		return
	}

	items := make([]RoleResponse, 0, len(userRoles))
	for _, ur := range userRoles {
		count, _ := h.roleRepo.CountUsersWithRole(c.Request.Context(), ur.RoleID)
		items = append(items, toRoleResponse(&ur.Role, count))
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}
