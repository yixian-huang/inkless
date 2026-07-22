package installed_theme

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// Handler handles installed-theme-related HTTP requests
type Handler struct {
	themeRepo        repository.InstalledThemeRepository
	themePageService *service.ThemePageService
	unifiedPageRepo  repository.UnifiedPageRepository
	cache            *cache.Cache
}

// NewHandler creates a new installed theme handler
func NewHandler(
	themeRepo repository.InstalledThemeRepository,
	themePageService *service.ThemePageService,
	c *cache.Cache,
	unifiedPageRepo repository.UnifiedPageRepository,
) *Handler {
	return &Handler{
		themeRepo:        themeRepo,
		themePageService: themePageService,
		unifiedPageRepo:  unifiedPageRepo,
		cache:            c,
	}
}

func (h *Handler) invalidateBootstrapCache() {
	cache.InvalidateThemeOrSiteConfig(h.cache)
}

// --- Public endpoints ---

// PublicGetActive returns the currently active theme.
// @Summary      Get active theme
// @Description  Returns the currently active installed theme
// @Tags         Themes
// @Produce      json
// @Success      200 {object} object{themeId=string,source=string,externalUrl=string}
// @Failure      404 {object} object{error=string}
// @Router       /public/active-theme [get]
func (h *Handler) PublicGetActive(c *gin.Context) {
	theme, err := h.themeRepo.FindActive(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "没有激活的主题")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"themeId":     theme.ThemeID,
		"source":      theme.Source,
		"externalUrl": theme.ExternalURL,
	})
}

// --- Admin endpoints ---

// AdminList returns all installed themes.
// @Summary      List installed themes
// @Description  Returns all installed themes
// @Tags         Themes
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} object
// @Router       /admin/themes [get]
func (h *Handler) AdminList(c *gin.Context) {
	themes, err := h.themeRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询主题失败")
		return
	}

	c.JSON(http.StatusOK, themes)
}

// AdminGetByID returns a single installed theme by ID.
// @Summary      Get theme by ID
// @Description  Returns a single installed theme
// @Tags         Themes
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Theme ID"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/themes/{id} [get]
func (h *Handler) AdminGetByID(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Use themeID lookup via list + filter since repo exposes FindByThemeID
	themes, err := h.themeRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询主题失败")
		return
	}

	for _, t := range themes {
		if t.ID == id {
			c.JSON(http.StatusOK, t)
			return
		}
	}

	apierror.Message(c, http.StatusNotFound, "主题不存在")
}

// createInput is the JSON body for installing an external theme
type createInput struct {
	ThemeID     string        `json:"themeId"`
	Name        string        `json:"name"`
	NameZh      string        `json:"nameZh"`
	Description string        `json:"description"`
	Author      string        `json:"author"`
	Version     string        `json:"version"`
	Source      string        `json:"source"`
	ExternalURL string        `json:"externalUrl"`
	Preview     string        `json:"preview"`
	Config      model.JSONMap `json:"config"`
}

// AdminCreate installs a new external theme.
// @Summary      Install theme
// @Description  Install a new external theme
// @Tags         Themes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Theme data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/themes [post]
func (h *Handler) AdminCreate(c *gin.Context) {
	var input createInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if input.ThemeID == "" {
		apierror.Message(c, http.StatusBadRequest, "themeId 不能为空")
		return
	}
	if input.Name == "" {
		apierror.Message(c, http.StatusBadRequest, "name 不能为空")
		return
	}

	source := input.Source
	if source == "" {
		source = "external"
	}

	theme := &model.InstalledTheme{
		ThemeID:     input.ThemeID,
		Name:        input.Name,
		NameZh:      input.NameZh,
		Description: input.Description,
		Author:      input.Author,
		Version:     input.Version,
		Source:      source,
		ExternalURL: input.ExternalURL,
		Preview:     input.Preview,
		Config:      input.Config,
	}

	if err := h.themeRepo.Create(c.Request.Context(), theme); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, theme)
}

// updateInput is the JSON body for updating a theme's configuration
type updateInput struct {
	Name        string        `json:"name"`
	NameZh      string        `json:"nameZh"`
	Description string        `json:"description"`
	Author      string        `json:"author"`
	Version     string        `json:"version"`
	ExternalURL string        `json:"externalUrl"`
	Preview     string        `json:"preview"`
	Config      model.JSONMap `json:"config"`
}

// AdminUpdate updates an installed theme's configuration.
// @Summary      Update theme
// @Description  Update an installed theme's configuration
// @Tags         Themes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Theme ID"
// @Param        body body object true "Updated theme data"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/themes/{id} [put]
func (h *Handler) AdminUpdate(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Find existing theme by iterating over all themes
	themes, err := h.themeRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询主题失败")
		return
	}

	var existing *model.InstalledTheme
	for _, t := range themes {
		if t.ID == id {
			existing = t
			break
		}
	}
	if existing == nil {
		apierror.Message(c, http.StatusNotFound, "主题不存在")
		return
	}

	var input updateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.NameZh != "" {
		existing.NameZh = input.NameZh
	}
	if input.Description != "" {
		existing.Description = input.Description
	}
	if input.Author != "" {
		existing.Author = input.Author
	}
	if input.Version != "" {
		existing.Version = input.Version
	}
	if input.ExternalURL != "" {
		existing.ExternalURL = input.ExternalURL
	}
	if input.Preview != "" {
		existing.Preview = input.Preview
	}
	if input.Config != nil {
		existing.Config = input.Config
	}

	if err := h.themeRepo.Update(c.Request.Context(), existing); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	if existing.IsActive && input.Config != nil {
		h.invalidateBootstrapCache()
	}

	c.JSON(http.StatusOK, existing)
}

// AdminDelete uninstalls a theme.
// @Summary      Delete theme
// @Description  Uninstall (soft-delete) a theme
// @Tags         Themes
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Theme ID"
// @Success      200 {object} object{message=string}
// @Failure      400 {object} object{error=string}
// @Router       /admin/themes/{id} [delete]
func (h *Handler) AdminDelete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Find existing theme to check constraints
	themes, err := h.themeRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询主题失败")
		return
	}

	var existing *model.InstalledTheme
	for _, t := range themes {
		if t.ID == id {
			existing = t
			break
		}
	}
	if existing == nil {
		apierror.Message(c, http.StatusNotFound, "主题不存在")
		return
	}

	// Cannot delete active theme
	if existing.IsActive {
		apierror.Message(c, http.StatusBadRequest, "不能删除当前激活的主题")
		return
	}

	// Cannot delete built-in theme
	if existing.Source == "built-in" {
		apierror.Message(c, http.StatusBadRequest, "不能删除内置主题")
		return
	}

	if err := h.themeRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "主题不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// AdminActivate activates a theme.
// @Summary      Activate theme
// @Description  Set a theme as the active theme and seed its pages
// @Tags         Themes
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Theme ID"
// @Success      200 {object} object
// @Router       /admin/themes/{id}/activate [put]
func (h *Handler) AdminActivate(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Find the theme to get its themeID
	themes, err := h.themeRepo.List(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询主题失败")
		return
	}

	var target *model.InstalledTheme
	for _, t := range themes {
		if t.ID == id {
			target = t
			break
		}
	}
	if target == nil {
		apierror.Message(c, http.StatusNotFound, "主题不存在")
		return
	}

	if err := h.themeRepo.SetActive(c.Request.Context(), target.ThemeID); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "激活主题失败")
		return
	}

	h.invalidateBootstrapCache()

	// Seed theme pages for the newly activated theme
	if h.themePageService != nil {
		if err := h.themePageService.SeedThemePages(c.Request.Context(), target.ThemeID); err != nil {
			// Log but don't fail the activation
			c.JSON(http.StatusOK, gin.H{
				"theme":   target,
				"warning": "主题已激活，但页面同步失败: " + err.Error(),
			})
			return
		}
	}

	// editorial-firm: fill empty unified page section configs (never overwrite existing)
	if target.ThemeID == builtinthemes.EditorialFirm && h.unifiedPageRepo != nil {
		if err := service.ApplyEditorialFirmPageSeeds(c.Request.Context(), h.unifiedPageRepo); err != nil {
			// Warning only — activation must still succeed
			log.Printf("Warning: editorial-firm unified page seeds failed: %v", err)
		}
	}

	// Return updated theme
	target.IsActive = true
	c.JSON(http.StatusOK, target)
}
