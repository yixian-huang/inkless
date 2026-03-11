package menu

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
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

// ListGroups returns all menu groups
// GET /admin/menus
func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.menuRepo.ListGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询菜单失败"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": groups})
}

// CreateGroup creates a new menu group
// POST /admin/menus
func (h *Handler) CreateGroup(c *gin.Context) {
	var input model.MenuGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	if err := h.menuRepo.CreateGroup(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// GetGroup returns a menu group with its items (flat list)
// GET /admin/menus/:id
func (h *Handler) GetGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	group, err := h.menuRepo.FindGroupByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "菜单不存在"}})
		return
	}

	c.JSON(http.StatusOK, group)
}

// UpdateGroup updates a menu group
// PUT /admin/menus/:id
func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	existing, err := h.menuRepo.FindGroupByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "菜单不存在"}})
		return
	}

	var input model.MenuGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	existing.Name = input.Name
	existing.Slug = input.Slug

	if err := h.menuRepo.UpdateGroup(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteGroup deletes a menu group
// DELETE /admin/menus/:id
func (h *Handler) DeleteGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.menuRepo.DeleteGroup(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "菜单不存在"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// SetPrimary sets a menu group as the primary menu
// PUT /admin/menus/:id/primary
func (h *Handler) SetPrimary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.menuRepo.SetPrimary(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已设置为主菜单"})
}

// CreateItem creates a new menu item in a group
// POST /admin/menus/:id/items
func (h *Handler) CreateItem(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	// Verify group exists
	if _, err := h.menuRepo.FindGroupByID(c.Request.Context(), uint(groupID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "菜单不存在"}})
		return
	}

	var input model.MenuItem
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	input.GroupID = uint(groupID)

	if err := h.menuRepo.CreateItem(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// UpdateItem updates a menu item
// PUT /admin/menus/:id/items/:itemId
func (h *Handler) UpdateItem(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	existing, err := h.menuRepo.FindItemByID(c.Request.Context(), uint(itemID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "菜单项不存在"}})
		return
	}

	var input model.MenuItem
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteItem deletes a menu item
// DELETE /admin/menus/:id/items/:itemId
func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	if err := h.menuRepo.DeleteItem(c.Request.Context(), uint(itemID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "菜单项不存在"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ReorderItems reorders menu items in a group
// PUT /admin/menus/:id/items/reorder
func (h *Handler) ReorderItems(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	var input struct {
		ItemIDs []uint `json:"itemIds"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的请求数据"}})
		return
	}

	if err := h.menuRepo.ReorderItems(c.Request.Context(), uint(groupID), input.ItemIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "排序失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "排序已更新"})
}

// --- Public endpoints ---

// PublicGetPrimary returns the primary menu group with items built into tree structure
// GET /public/menu
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
