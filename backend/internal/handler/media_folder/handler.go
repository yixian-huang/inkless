package media_folder

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// Handler handles media folder HTTP requests
type Handler struct {
	folderRepo repository.MediaFolderRepository
	mediaRepo  repository.MediaRepository
}

// NewHandler creates a new media folder handler
func NewHandler(folderRepo repository.MediaFolderRepository, mediaRepo repository.MediaRepository) *Handler {
	return &Handler{
		folderRepo: folderRepo,
		mediaRepo:  mediaRepo,
	}
}

// ListTree returns the folder tree
// GET /admin/media/folders
func (h *Handler) ListTree(c *gin.Context) {
	folders, err := h.folderRepo.ListAll(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "获取文件夹列表失败")
		return
	}

	tree := buildTree(folders)
	c.JSON(http.StatusOK, gin.H{"folders": tree})
}

// CreateRequest is the request body for creating a folder
type CreateRequest struct {
	Name     string `json:"name" binding:"required"`
	ParentID *uint  `json:"parentId"`
}

// Create creates a new media folder
// POST /admin/media/folders
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	// Build materialized path
	path := "/" + req.Name
	if req.ParentID != nil {
		parent, err := h.folderRepo.FindByID(c.Request.Context(), *req.ParentID)
		if err != nil {
			apierror.Message(c, http.StatusBadRequest, "父文件夹不存在")
			return
		}
		path = parent.Path + "/" + req.Name
	}

	folder := &model.MediaFolder{
		Name:     req.Name,
		ParentID: req.ParentID,
		Path:     path,
	}

	if err := h.folderRepo.Create(c.Request.Context(), folder); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, folder)
}

// RenameRequest is the request body for renaming a folder
type RenameRequest struct {
	Name string `json:"name" binding:"required"`
}

// Rename renames a media folder
// PUT /admin/media/folders/:id
func (h *Handler) Rename(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	folder, err := h.folderRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "文件夹不存在")
		return
	}

	// Update name and recalculate path
	folder.Name = req.Name
	if folder.ParentID != nil {
		parent, err := h.folderRepo.FindByID(c.Request.Context(), *folder.ParentID)
		if err == nil {
			folder.Path = parent.Path + "/" + req.Name
		}
	} else {
		folder.Path = "/" + req.Name
	}

	if err := h.folderRepo.Update(c.Request.Context(), folder); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "更新文件夹失败")
		return
	}

	c.JSON(http.StatusOK, folder)
}

// Delete deletes a media folder and moves its contents to the parent folder
// DELETE /admin/media/folders/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	folder, err := h.folderRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "文件夹不存在")
		return
	}

	// Move child folders to parent
	children, err := h.folderRepo.FindChildren(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "获取子文件夹失败")
		return
	}

	for _, child := range children {
		child.ParentID = folder.ParentID
		if folder.ParentID != nil {
			parent, err := h.folderRepo.FindByID(c.Request.Context(), *folder.ParentID)
			if err == nil {
				child.Path = parent.Path + "/" + child.Name
			}
		} else {
			child.Path = "/" + child.Name
		}
		if err := h.folderRepo.Update(c.Request.Context(), child); err != nil {
			apierror.Message(c, http.StatusInternalServerError, fmt.Sprintf("移动子文件夹失败: %v", err))
			return
		}
	}

	// Delete the folder
	if err := h.folderRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "删除文件夹失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件夹已删除"})
}

// MoveMediaRequest is the request body for moving media to a folder
type MoveMediaRequest struct {
	FolderID *uint `json:"folderId"`
}

// MoveMedia moves a media item to a folder
// PUT /admin/media/:id/move
func (h *Handler) MoveMedia(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var req MoveMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	// Verify media exists
	if _, err := h.mediaRepo.FindByID(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "媒体文件不存在")
		return
	}

	// Verify target folder exists (if specified)
	if req.FolderID != nil {
		if _, err := h.folderRepo.FindByID(c.Request.Context(), *req.FolderID); err != nil {
			apierror.Message(c, http.StatusBadRequest, "目标文件夹不存在")
			return
		}
	}

	if err := h.folderRepo.UpdateMediaFolder(c.Request.Context(), id, req.FolderID); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "移动媒体文件失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "媒体文件已移动"})
}

// folderNode is used to build the folder tree response
type folderNode struct {
	ID       uint          `json:"id"`
	Name     string        `json:"name"`
	ParentID *uint         `json:"parentId,omitempty"`
	Path     string        `json:"path"`
	Children []*folderNode `json:"children"`
}

// buildTree constructs a tree structure from a flat list of folders
func buildTree(folders []*model.MediaFolder) []*folderNode {
	nodeMap := make(map[uint]*folderNode)
	var roots []*folderNode

	// Create nodes
	for _, f := range folders {
		nodeMap[f.ID] = &folderNode{
			ID:       f.ID,
			Name:     f.Name,
			ParentID: f.ParentID,
			Path:     f.Path,
			Children: []*folderNode{},
		}
	}

	// Build tree
	for _, f := range folders {
		node := nodeMap[f.ID]
		if f.ParentID == nil {
			roots = append(roots, node)
		} else {
			if parent, ok := nodeMap[*f.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				// Orphan - add to roots
				roots = append(roots, node)
			}
		}
	}

	if roots == nil {
		roots = []*folderNode{}
	}

	return roots
}
