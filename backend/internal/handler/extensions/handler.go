package extensions

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// Handler serves Phase A official extension-store endpoints (theme catalog first).
type Handler struct {
	loader           *themecatalog.Loader
	themeRepo        repository.InstalledThemeRepository
	themePageService *service.ThemePageService
	publicCache      *cache.Cache
	hostVersion      string
	allowHosts       []string
	builtinIDs       map[string]struct{}
}

// NewHandler wires catalog loader + installed themes for the admin market API.
func NewHandler(
	loader *themecatalog.Loader,
	themeRepo repository.InstalledThemeRepository,
	hostVersion string,
	allowHosts []string,
) *Handler {
	if loader == nil {
		loader = themecatalog.NewLoader("")
	}
	return &Handler{
		loader:      loader,
		themeRepo:   themeRepo,
		hostVersion: strings.TrimSpace(hostVersion),
		allowHosts:  allowHosts,
		builtinIDs: themecatalog.BuiltinIDSet(
			builtinthemes.CorporateClassic,
			builtinthemes.BlogFirst,
			builtinthemes.ProductFirst,
			builtinthemes.MinimalStarter,
			builtinthemes.EditorialFirm,
		),
	}
}

// WithActivation enables install-and-activate (SetActive + SeedThemePages + bootstrap invalidate).
func (h *Handler) WithActivation(themePageService *service.ThemePageService, publicCache *cache.Cache) *Handler {
	if h == nil {
		return nil
	}
	h.themePageService = themePageService
	h.publicCache = publicCache
	return h
}

// AdminThemeCatalog godoc
// @Summary      Official theme catalog
// @Description  Lists official themes with installState merged from this instance
// @Tags         Extensions
// @Produce      json
// @Security     BearerAuth
// @Param        refresh query bool false "Bypass catalog cache and re-fetch remote index"
// @Success      200 {object} object
// @Router       /admin/extensions/themes/catalog [get]
func (h *Handler) AdminThemeCatalog(c *gin.Context) {
	refresh := parseRefreshQuery(c.Query("refresh"))

	loadRes, err := h.loader.Load(c.Request.Context(), refresh)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "加载主题目录失败")
		return
	}

	var installedRefs []themecatalog.InstalledRef
	if h.themeRepo != nil {
		themes, listErr := h.themeRepo.List(c.Request.Context())
		if listErr != nil {
			apierror.Message(c, http.StatusInternalServerError, "查询已安装主题失败")
			return
		}
		installedRefs = make([]themecatalog.InstalledRef, 0, len(themes))
		for _, t := range themes {
			if t == nil {
				continue
			}
			installedRefs = append(installedRefs, themecatalog.InstalledRef{
				ThemeID:     t.ThemeID,
				Version:     t.Version,
				Source:      t.Source,
				ExternalURL: t.ExternalURL,
				IsActive:    t.IsActive,
			})
		}
	}

	items := themecatalog.MergeCatalogStatuses(loadRes.Catalog, installedRefs, themecatalog.MergeOptions{
		BuiltinIDs:         h.builtinIDs,
		SupportedContracts: themecatalog.HostSupportedContracts,
		HostVersion:        h.hostVersion,
		AllowHosts:         h.allowHosts,
	})

	// Flatten entry fields for admin UI convenience while keeping status fields.
	flat := make([]gin.H, 0, len(items))
	for _, it := range items {
		e := it.Entry
		flat = append(flat, gin.H{
			"slug":               e.Slug,
			"themeId":            e.ThemeID,
			"name":               e.Name,
			"nameZh":             e.NameZh,
			"description":        e.Description,
			"descriptionZh":      e.DescriptionZh,
			"author":             e.Author,
			"category":           e.Category,
			"tags":               e.Tags,
			"iconUrl":            e.IconURL,
			"previewUrl":         e.PreviewURL,
			"repoUrl":            e.RepoURL,
			"contractVersion":    e.ContractVersion,
			"minHostVersion":     e.MinHostVersion,
			"latest":             e.Latest,
			"versions":           e.Versions,
			"defaultFeaturesHint": e.DefaultFeaturesHint,
			"builtinOnly":        e.BuiltinOnly,
			"official":           e.Official,
			"installState":       it.InstallState,
			"installedVersion":   it.InstalledVersion,
			"installedSource":    it.InstalledSource,
			"incompatibleReason": emptyToNil(it.IncompatibleReason),
			"updateAvailable":    it.UpdateAvailable,
		})
	}

	updatedAt := ""
	schemaVersion := 1
	if loadRes.Catalog != nil {
		updatedAt = loadRes.Catalog.UpdatedAt
		schemaVersion = loadRes.Catalog.SchemaVersion
	}

	c.JSON(http.StatusOK, gin.H{
		"schemaVersion": schemaVersion,
		"source":        loadRes.Source,
		"updatedAt":     updatedAt,
		"warning":       emptyToNil(loadRes.Warning),
		"items":         flat,
	})
}

type installThemeInput struct {
	Slug     string `json:"slug"`
	Version  string `json:"version"`  // optional; empty / "latest" → catalog latest
	Activate bool   `json:"activate"` // optional; requires activation deps when true
}

// AdminThemeInstall godoc
// @Summary      Install official theme from catalog
// @Description  Validates catalog entry and upserts installed_themes (source=marketplace for UMD themes)
// @Tags         Extensions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body installThemeInput true "Install request"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/extensions/themes/install [post]
func (h *Handler) AdminThemeInstall(c *gin.Context) {
	var input installThemeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		apierror.Message(c, http.StatusBadRequest, "slug 不能为空")
		return
	}
	if h.themeRepo == nil {
		apierror.Message(c, http.StatusInternalServerError, "主题仓库未配置")
		return
	}

	loadRes, err := h.loader.Load(c.Request.Context(), false)
	if err != nil || loadRes == nil || loadRes.Catalog == nil {
		apierror.Message(c, http.StatusInternalServerError, "加载主题目录失败")
		return
	}

	entry, ok := loadRes.Catalog.FindBySlug(slug)
	if !ok {
		apierror.Message(c, http.StatusNotFound, "主题不在官方目录中")
		return
	}

	ver, err := entry.ResolveVersion(input.Version)
	if err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	// Full installability: official + contract + minHost + allowlist (UMD when needed).
	if reason := installBlockReason(entry, ver, h.hostVersion, h.allowHosts); reason != "" {
		apierror.Message(c, http.StatusBadRequest, reason)
		return
	}

	theme, created, err := h.upsertInstalledFromCatalog(c, entry, ver)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "安装主题失败: "+err.Error())
		return
	}

	activated := false
	var activateWarning string
	if input.Activate {
		if actErr := h.activateInstalled(c, theme.ThemeID); actErr != nil {
			activateWarning = actErr.Error()
		} else {
			activated = true
			// Refresh after activate
			if refreshed, findErr := h.themeRepo.FindByThemeID(c.Request.Context(), theme.ThemeID); findErr == nil && refreshed != nil {
				theme = refreshed
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"theme":     theme,
		"created":   created,
		"activated": activated,
		"warning":   emptyToNil(activateWarning),
	})
}

func (h *Handler) upsertInstalledFromCatalog(
	c *gin.Context,
	entry *themecatalog.ThemeEntry,
	ver *themecatalog.ThemeVersion,
) (*model.InstalledTheme, bool, error) {
	ctx := c.Request.Context()
	existing, err := h.themeRepo.FindByThemeID(ctx, entry.ThemeID)
	notFound := err != nil && strings.Contains(err.Error(), "not found")
	if err != nil && !notFound {
		return nil, false, err
	}
	if notFound {
		existing = nil
	}

	source, externalURL := installSourceAndURL(entry, ver, existing)

	if existing == nil {
		theme := &model.InstalledTheme{
			ThemeID:     entry.ThemeID,
			Name:        entry.Name,
			NameZh:      entry.NameZh,
			Description: pickDescription(entry),
			Author:      entry.Author,
			Version:     ver.Version,
			Source:      source,
			ExternalURL: externalURL,
			Preview:     entry.PreviewURL,
			IsActive:    false,
			Config:      model.JSONMap{},
		}
		if err := h.themeRepo.Create(ctx, theme); err != nil {
			return nil, false, err
		}
		return theme, true, nil
	}

	// Update metadata / package pointer; preserve user Config and IsActive.
	existing.Name = entry.Name
	if strings.TrimSpace(entry.NameZh) != "" {
		existing.NameZh = entry.NameZh
	}
	existing.Description = pickDescription(entry)
	if strings.TrimSpace(entry.Author) != "" {
		existing.Author = entry.Author
	}
	existing.Version = ver.Version
	existing.Source = source
	existing.ExternalURL = externalURL
	if strings.TrimSpace(entry.PreviewURL) != "" {
		existing.Preview = entry.PreviewURL
	}
	if err := h.themeRepo.Update(ctx, existing); err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (h *Handler) activateInstalled(c *gin.Context, themeID string) error {
	if err := h.themeRepo.SetActive(c.Request.Context(), themeID); err != nil {
		return err
	}
	cache.InvalidateThemeOrSiteConfig(h.publicCache)
	if h.themePageService != nil {
		if err := h.themePageService.SeedThemePages(c.Request.Context(), themeID); err != nil {
			// Match AdminActivate: activation succeeds with page seed warning.
			log.Printf("Warning: seed theme pages after marketplace install: %v", err)
			return errors.New("主题已激活，但页面同步失败: " + err.Error())
		}
	}
	return nil
}

func installSourceAndURL(
	entry *themecatalog.ThemeEntry,
	ver *themecatalog.ThemeVersion,
	existing *model.InstalledTheme,
) (source, externalURL string) {
	if entry.BuiltinOnly || strings.TrimSpace(ver.UMDURL) == "" {
		// Keep built-in rows as built-in; new rows default built-in.
		if existing != nil && (existing.Source == "built-in" || existing.Source == "builtin") {
			return existing.Source, ""
		}
		return "built-in", ""
	}
	return "marketplace", strings.TrimSpace(ver.UMDURL)
}

func pickDescription(entry *themecatalog.ThemeEntry) string {
	if entry == nil {
		return ""
	}
	if strings.TrimSpace(entry.Description) != "" {
		return entry.Description
	}
	return entry.DescriptionZh
}

// installBlockReason returns a user-facing error if the catalog version cannot be installed.
func installBlockReason(
	entry *themecatalog.ThemeEntry,
	ver *themecatalog.ThemeVersion,
	hostVersion string,
	allowHosts []string,
) string {
	if entry == nil || ver == nil {
		return "无效的主题条目"
	}
	if !entry.Official {
		return "仅支持安装官方主题"
	}
	// Reuse merge incompatible checks against the entry's catalog metadata,
	// then ValidateEntryInstallable for the resolved version UMD.
	st := themecatalog.MergeInstallState(*entry, nil, themecatalog.MergeOptions{
		SupportedContracts: themecatalog.HostSupportedContracts,
		HostVersion:        hostVersion,
		AllowHosts:         allowHosts,
	})
	if st.InstallState == themecatalog.InstallStateIncompatible {
		// Allow install when only reason was "not installed" path — incompatible is hard block.
		// For builtinOnly, MergeInstallState won't flag UMD; good.
		if st.IncompatibleReason != "" {
			return st.IncompatibleReason
		}
		return "主题与当前实例不兼容"
	}
	if err := themecatalog.ValidateEntryInstallable(entry, ver, allowHosts); err != nil {
		return err.Error()
	}
	return ""
}

func parseRefreshQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func emptyToNil(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
