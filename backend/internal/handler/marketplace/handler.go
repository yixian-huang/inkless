package marketplace

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// Handler handles marketplace HTTP requests
type Handler struct {
	svc *service.MarketplaceService
}

// NewHandler creates a new marketplace Handler
func NewHandler(svc *service.MarketplaceService) *Handler {
	return &Handler{svc: svc}
}

// AdminListItems godoc
// @Summary      List marketplace items
// @Description  Returns a paginated list of marketplace items with optional filters
// @Tags         Marketplace
// @Produce      json
// @Security     BearerAuth
// @Param        type     query  string false "Item type: plugin or theme"
// @Param        category query  string false "Category filter"
// @Param        search   query  string false "Search term"
// @Param        page     query  int    false "Page number (default 1)"
// @Param        pageSize query  int    false "Page size (default 20, max 100)"
// @Success      200 {object} object
// @Router       /admin/marketplace/items [get]
func (h *Handler) AdminListItems(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 20, 100)
	page, pageSize := p.Page, p.PageSize

	filter := repository.MarketplaceFilter{
		Type:     c.Query("type"),
		Category: c.Query("category"),
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}

	items, total, err := h.svc.SearchItems(c.Request.Context(), filter)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询市场列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminGetItem godoc
// @Summary      Get marketplace item details
// @Description  Returns details for a marketplace item including all versions
// @Tags         Marketplace
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Item slug"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/marketplace/items/{slug} [get]
func (h *Handler) AdminGetItem(c *gin.Context) {
	slug := c.Param("slug")

	item, err := h.svc.GetItemDetails(c.Request.Context(), slug)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "市场项目不存在")
		return
	}

	c.JSON(http.StatusOK, item)
}

// AdminInstallItem godoc
// @Summary      Install a marketplace item
// @Description  Installs a plugin or theme from the marketplace (increments download count and returns download URL)
// @Tags         Marketplace
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Item slug"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/marketplace/items/{slug}/install [post]
func (h *Handler) AdminInstallItem(c *gin.Context) {
	slug := c.Param("slug")

	item, err := h.svc.InstallItem(c.Request.Context(), slug)
	if err != nil {
		if isNotFoundError(err) {
			apierror.Message(c, http.StatusNotFound, "市场项目不存在")
			return
		}
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item":        item,
		"downloadUrl": item.DownloadURL,
		"message":     "安装请求已受理，请使用 downloadUrl 下载安装包",
	})
}

// AdminUpdateItem godoc
// @Summary      Update an installed marketplace item
// @Description  Records an update download for an installed marketplace item
// @Tags         Marketplace
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Item slug"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/marketplace/items/{slug}/update [put]
func (h *Handler) AdminUpdateItem(c *gin.Context) {
	slug := c.Param("slug")

	item, err := h.svc.UpdateItem(c.Request.Context(), slug)
	if err != nil {
		if isNotFoundError(err) {
			apierror.Message(c, http.StatusNotFound, "市场项目不存在")
			return
		}
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item":        item,
		"downloadUrl": item.DownloadURL,
		"message":     "更新请求已受理，请使用 downloadUrl 下载更新包",
	})
}

// AdminListInstalled godoc
// @Summary      List installed marketplace items
// @Description  Returns all active marketplace items (proxy for installed state)
// @Tags         Marketplace
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} object
// @Router       /admin/marketplace/installed [get]
func (h *Handler) AdminListInstalled(c *gin.Context) {
	items, err := h.svc.ListInstalled(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询已安装项目失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// registerItemInput is the JSON body for registering a new marketplace item
type registerItemInput struct {
	Type        model.MarketplaceItemType   `json:"type"`
	Name        string                      `json:"name"`
	NameZh      string                      `json:"nameZh"`
	Slug        string                      `json:"slug"`
	Description string                      `json:"description"`
	Author      string                      `json:"author"`
	Version     string                      `json:"version"`
	IconURL     string                      `json:"iconUrl"`
	PreviewURL  string                      `json:"previewUrl"`
	DownloadURL string                      `json:"downloadUrl"`
	Category    string                      `json:"category"`
	Tags        model.JSONStringSlice       `json:"tags"`
	Status      model.MarketplaceItemStatus `json:"status"`
}

// AdminRegisterItem godoc
// @Summary      Register a marketplace item
// @Description  Adds a new plugin or theme to the marketplace registry
// @Tags         Marketplace
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Item data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/marketplace/items [post]
func (h *Handler) AdminRegisterItem(c *gin.Context) {
	var input registerItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	item := &model.MarketplaceItem{
		Type:        input.Type,
		Name:        input.Name,
		NameZh:      input.NameZh,
		Slug:        input.Slug,
		Description: input.Description,
		Author:      input.Author,
		Version:     input.Version,
		IconURL:     input.IconURL,
		PreviewURL:  input.PreviewURL,
		DownloadURL: input.DownloadURL,
		Category:    input.Category,
		Tags:        input.Tags,
		Status:      input.Status,
	}

	if err := h.svc.RegisterItem(c.Request.Context(), item); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, item)
}

// AdminUninstallItem godoc
// @Summary      Uninstall/remove a marketplace item
// @Description  Soft-deletes a marketplace item from the registry
// @Tags         Marketplace
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Item slug"
// @Success      200 {object} object{message=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/marketplace/items/{slug} [delete]
func (h *Handler) AdminUninstallItem(c *gin.Context) {
	slug := c.Param("slug")

	if err := h.svc.UninstallItem(c.Request.Context(), slug); err != nil {
		if isNotFoundError(err) {
			apierror.Message(c, http.StatusNotFound, "市场项目不存在")
			return
		}
		apierror.Message(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已卸载"})
}

// addVersionInput is the JSON body for adding a version to a marketplace item
type addVersionInput struct {
	Version       string `json:"version"`
	Changelog     string `json:"changelog"`
	DownloadURL   string `json:"downloadUrl"`
	MinAppVersion string `json:"minAppVersion"`
}

// AdminAddVersion godoc
// @Summary      Add a version to a marketplace item
// @Description  Records a new version for a marketplace item
// @Tags         Marketplace
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Item slug"
// @Param        body body object true "Version data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/marketplace/items/{slug}/versions [post]
func (h *Handler) AdminAddVersion(c *gin.Context) {
	slug := c.Param("slug")

	var input addVersionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if input.Version == "" {
		apierror.Message(c, http.StatusBadRequest, "版本号不能为空")
		return
	}

	version := &model.MarketplaceVersion{
		Version:       input.Version,
		Changelog:     input.Changelog,
		DownloadURL:   input.DownloadURL,
		MinAppVersion: input.MinAppVersion,
	}

	if err := h.svc.AddVersion(c.Request.Context(), slug, version); err != nil {
		if isNotFoundError(err) {
			apierror.Message(c, http.StatusNotFound, "市场项目不存在")
			return
		}
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, version)
}

// isNotFoundError is a simple heuristic for detecting not-found errors from the service layer.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) > 0 && (contains(msg, "not found") || contains(msg, "不存在"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchString(s, substr))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
