package menu

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// Handler handles menu-related HTTP requests
type Handler struct {
	menuRepo repository.MenuRepository
}

// NewHandler creates a new menu handler
func NewHandler(menuRepo repository.MenuRepository) *Handler {
	return &Handler{menuRepo: menuRepo}
}

// --- Admin endpoints ---

// ListGroups returns all menu groups.
// @Summary      List menu groups
// @Description  Returns all menu groups
// @Tags         Menus
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{items=[]object}
// @Router       /admin/menus [get]
func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.menuRepo.ListGroups(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询菜单失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": groups})
}

// CreateGroup creates a new menu group.
// @Summary      Create menu group
// @Description  Create a new menu group
// @Tags         Menus
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Menu group data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/menus [post]
func (h *Handler) CreateGroup(c *gin.Context) {
	var input model.MenuGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if err := h.menuRepo.CreateGroup(c.Request.Context(), &input); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, input)
}

// GetGroup returns a menu group with its items.
// @Summary      Get menu group
// @Description  Returns a menu group with its items as a flat list
// @Tags         Menus
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Menu group ID"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/menus/{id} [get]
func (h *Handler) GetGroup(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	group, err := h.menuRepo.FindGroupByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "菜单不存在")
		return
	}

	c.JSON(http.StatusOK, group)
}

// UpdateGroup updates a menu group.
// @Summary      Update menu group
// @Description  Update a menu group's name and slug
// @Tags         Menus
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Menu group ID"
// @Param        body body object true "Updated menu group data"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/menus/{id} [put]
func (h *Handler) UpdateGroup(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	existing, err := h.menuRepo.FindGroupByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "菜单不存在")
		return
	}

	var input model.MenuGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	existing.Name = input.Name
	existing.Slug = input.Slug

	if err := h.menuRepo.UpdateGroup(c.Request.Context(), existing); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteGroup deletes a menu group.
// @Summary      Delete menu group
// @Description  Delete a menu group and its items
// @Tags         Menus
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Menu group ID"
// @Success      200 {object} object{message=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/menus/{id} [delete]
func (h *Handler) DeleteGroup(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.menuRepo.DeleteGroup(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "菜单不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// SetPrimary sets a menu group as the primary menu.
// @Summary      Set primary menu
// @Description  Set a menu group as the primary navigation menu
// @Tags         Menus
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Menu group ID"
// @Success      200 {object} object{message=string}
// @Router       /admin/menus/{id}/primary [put]
func (h *Handler) SetPrimary(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.menuRepo.SetPrimary(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已设置为主菜单"})
}

// CreateItem creates a new menu item in a group.
// @Summary      Create menu item
// @Description  Create a new menu item within a menu group
// @Tags         Menus
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Menu group ID"
// @Param        body body object true "Menu item data"
// @Success      201 {object} object
// @Router       /admin/menus/{id}/items [post]
func (h *Handler) CreateItem(c *gin.Context) {
	groupID, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Verify group exists
	if _, err := h.menuRepo.FindGroupByID(c.Request.Context(), groupID); err != nil {
		apierror.Message(c, http.StatusNotFound, "菜单不存在")
		return
	}

	var input model.MenuItem
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	input.GroupID = groupID

	if err := h.menuRepo.CreateItem(c.Request.Context(), &input); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, input)
}

// UpdateItem updates a menu item.
// @Summary      Update menu item
// @Description  Update a menu item's properties
// @Tags         Menus
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id     path int    true "Menu group ID"
// @Param        itemId path int    true "Menu item ID"
// @Param        body   body object true "Updated menu item data"
// @Success      200 {object} object
// @Router       /admin/menus/{id}/items/{itemId} [put]
func (h *Handler) UpdateItem(c *gin.Context) {
	itemID, ok := handlerutil.ParseUintParam(c, "itemId")
	if !ok {
		return
	}

	existing, err := h.menuRepo.FindItemByID(c.Request.Context(), itemID)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "菜单项不存在")
		return
	}

	var input model.MenuItem
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	existing.ParentID = input.ParentID
	existing.ZhName = input.ZhName
	existing.EnName = input.EnName
	existing.Type = input.Type
	existing.Target = input.Target
	existing.URL = input.URL
	existing.RefID = input.RefID
	existing.RefSlug = input.RefSlug
	existing.Visible = input.Visible
	existing.Metadata = input.Metadata
	existing.SortOrder = input.SortOrder

	if err := h.menuRepo.UpdateItem(c.Request.Context(), existing); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteItem deletes a menu item.
// @Summary      Delete menu item
// @Description  Delete a menu item by ID
// @Tags         Menus
// @Produce      json
// @Security     BearerAuth
// @Param        id     path int true "Menu group ID"
// @Param        itemId path int true "Menu item ID"
// @Success      200 {object} object{message=string}
// @Router       /admin/menus/{id}/items/{itemId} [delete]
func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, ok := handlerutil.ParseUintParam(c, "itemId")
	if !ok {
		return
	}

	if err := h.menuRepo.DeleteItem(c.Request.Context(), itemID); err != nil {
		apierror.Message(c, http.StatusNotFound, "菜单项不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ReorderItems reorders menu items in a group.
// @Summary      Reorder menu items
// @Description  Set the display order of menu items in a group
// @Tags         Menus
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Menu group ID"
// @Param        body body object true "Ordered item IDs"
// @Success      200 {object} object{message=string}
// @Router       /admin/menus/{id}/items/reorder [put]
func (h *Handler) ReorderItems(c *gin.Context) {
	groupID, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var input struct {
		ItemIDs []uint `json:"itemIds"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if err := h.menuRepo.ReorderItems(c.Request.Context(), groupID, input.ItemIDs); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "排序失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "排序已更新"})
}

// --- Public endpoints ---

// PublicGetPrimary returns the primary menu as a tree.
// @Summary      Get primary menu
// @Description  Returns the primary menu group with items as a nested tree
// @Tags         Menus
// @Produce      json
// @Success      200 {object} object{id=int,name=string,items=[]object}
// @Router       /public/menu [get]
func (h *Handler) PublicGetPrimary(c *gin.Context) {
	group, err := h.menuRepo.FindPrimaryGroup(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
		return
	}

	// Build tree from flat items
	items := buildMenuTree(group.Items)

	c.JSON(http.StatusOK, gin.H{
		"id":        group.ID,
		"name":      group.Name,
		"slug":      group.Slug,
		"isPrimary": group.IsPrimary,
		"items":     items,
	})
}

// buildMenuTree converts a flat list of menu items into a tree by parentId
func buildMenuTree(items []model.MenuItem) []model.MenuItem {
	byID := make(map[uint]*model.MenuItem)
	var roots []model.MenuItem

	// First pass: index all items
	for i := range items {
		items[i].Children = nil // reset
		byID[items[i].ID] = &items[i]
	}

	// Second pass: build tree
	for i := range items {
		if items[i].ParentID == nil {
			roots = append(roots, items[i])
		} else if parent, ok := byID[*items[i].ParentID]; ok {
			parent.Children = append(parent.Children, items[i])
		}
	}

	// Update roots with children from byID (since roots are copies)
	for i := range roots {
		if ref, ok := byID[roots[i].ID]; ok {
			roots[i].Children = ref.Children
		}
	}

	return roots
}
