package extensions

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// Handler serves Phase A official extension-store endpoints.
type Handler struct {
	installer  *service.OfficialThemeInstaller
	autoUpdate *service.ThemeAutoUpdateService
	builtinIDs map[string]struct{}
}

// NewHandler wires catalog installer (+ optional auto-update).
func NewHandler(installer *service.OfficialThemeInstaller, autoUpdate *service.ThemeAutoUpdateService) *Handler {
	return &Handler{
		installer:  installer,
		autoUpdate: autoUpdate,
		builtinIDs: themecatalog.BuiltinIDSet(
			builtinthemes.CorporateClassic,
			builtinthemes.BlogFirst,
			builtinthemes.ProductFirst,
			builtinthemes.MinimalStarter,
			builtinthemes.EditorialFirm,
		),
	}
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
	if h.installer == nil {
		apierror.Message(c, http.StatusInternalServerError, "主题安装器未配置")
		return
	}
	refresh := parseRefreshQuery(c.Query("refresh"))
	loadRes, err := h.installer.Loader().Load(c.Request.Context(), refresh)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "加载主题目录失败")
		return
	}

	installedRefs, listErr := h.installer.ListInstalledRefs(c.Request.Context())
	if listErr != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询已安装主题失败")
		return
	}

	items := themecatalog.MergeCatalogStatuses(loadRes.Catalog, installedRefs, themecatalog.MergeOptions{
		BuiltinIDs:         h.builtinIDs,
		SupportedContracts: themecatalog.HostSupportedContracts,
		HostVersion:        h.installer.HostVersion(),
		AllowHosts:         h.installer.AllowHosts(),
	})

	flat := make([]gin.H, 0, len(items))
	for _, it := range items {
		e := it.Entry
		flat = append(flat, gin.H{
			"slug":                e.Slug,
			"themeId":             e.ThemeID,
			"name":                e.Name,
			"nameZh":              e.NameZh,
			"description":         e.Description,
			"descriptionZh":       e.DescriptionZh,
			"author":              e.Author,
			"category":            e.Category,
			"tags":                e.Tags,
			"iconUrl":             e.IconURL,
			"previewUrl":          e.PreviewURL,
			"repoUrl":             e.RepoURL,
			"contractVersion":     e.ContractVersion,
			"minHostVersion":      e.MinHostVersion,
			"latest":              e.Latest,
			"versions":            e.Versions,
			"defaultFeaturesHint": e.DefaultFeaturesHint,
			"builtinOnly":         e.BuiltinOnly,
			"official":            e.Official,
			"installState":        it.InstallState,
			"installedVersion":    it.InstalledVersion,
			"installedSource":     it.InstalledSource,
			"incompatibleReason":  emptyToNil(it.IncompatibleReason),
			"updateAvailable":     it.UpdateAvailable,
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
	Version  string `json:"version"`
	Activate bool   `json:"activate"`
}

// AdminThemeInstall godoc
// @Summary      Install official theme from catalog
// @Tags         Extensions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Router       /admin/extensions/themes/install [post]
func (h *Handler) AdminThemeInstall(c *gin.Context) {
	if h.installer == nil {
		apierror.Message(c, http.StatusInternalServerError, "主题安装器未配置")
		return
	}
	var input installThemeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}
	res, err := h.installer.Install(c.Request.Context(), service.ThemeInstallRequest{
		Slug:     input.Slug,
		Version:  input.Version,
		Activate: input.Activate,
	})
	if err != nil {
		if service.IsNotInCatalog(err) {
			apierror.Message(c, http.StatusNotFound, err.Error())
			return
		}
		// validation-ish errors → 400
		msg := err.Error()
		if strings.Contains(msg, "失败") && !strings.Contains(msg, "校验") && !strings.Contains(msg, "不兼容") && !strings.Contains(msg, "不能为空") && !strings.Contains(msg, "仅支持") && !strings.Contains(msg, "requires") && !strings.Contains(msg, "allowlist") && !strings.Contains(msg, "contract") {
			apierror.Message(c, http.StatusInternalServerError, "安装主题失败: "+msg)
			return
		}
		apierror.Message(c, http.StatusBadRequest, msg)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"theme":     res.Theme,
		"created":   res.Created,
		"activated": res.Activated,
		"warning":   emptyToNil(res.Warning),
	})
}

// AdminThemeAutoUpdateGet returns auto-update settings + last report.
// @Router       /admin/extensions/themes/auto-update [get]
func (h *Handler) AdminThemeAutoUpdateGet(c *gin.Context) {
	if h.autoUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自动更新服务未配置")
		return
	}
	settings, err := h.autoUpdate.GetSettings(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "读取自动更新配置失败")
		return
	}
	c.JSON(http.StatusOK, settings)
}

type autoUpdatePutInput struct {
	Enabled         *bool `json:"enabled"`
	IntervalMinutes *int  `json:"intervalMinutes"`
	OnlyMarketplace *bool `json:"onlyMarketplace"`
	IncludeActive   *bool `json:"includeActive"`
	OnlyActive      *bool `json:"onlyActive"`
}

// AdminThemeAutoUpdatePut updates auto-update settings.
// @Router       /admin/extensions/themes/auto-update [put]
func (h *Handler) AdminThemeAutoUpdatePut(c *gin.Context) {
	if h.autoUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自动更新服务未配置")
		return
	}
	var input autoUpdatePutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}
	cur, err := h.autoUpdate.GetSettings(c.Request.Context())
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "读取自动更新配置失败")
		return
	}
	if input.Enabled != nil {
		cur.Enabled = *input.Enabled
	}
	if input.IntervalMinutes != nil {
		cur.IntervalMinutes = *input.IntervalMinutes
	}
	if input.OnlyMarketplace != nil {
		cur.OnlyMarketplace = *input.OnlyMarketplace
	}
	if input.IncludeActive != nil {
		cur.IncludeActive = *input.IncludeActive
	}
	if input.OnlyActive != nil {
		cur.OnlyActive = *input.OnlyActive
	}
	saved, err := h.autoUpdate.SaveSettings(c.Request.Context(), cur)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "保存自动更新配置失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, saved)
}

type autoUpdateRunInput struct {
	// DryRun only reports available updates without writing.
	DryRun bool `json:"dryRun"`
}

// AdminThemeAutoUpdateRun checks catalog (and applies updates unless dryRun).
// Runs even when auto-update is disabled (manual trigger).
// @Router       /admin/extensions/themes/auto-update/run [post]
func (h *Handler) AdminThemeAutoUpdateRun(c *gin.Context) {
	if h.autoUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自动更新服务未配置")
		return
	}
	var input autoUpdateRunInput
	_ = c.ShouldBindJSON(&input)

	var (
		report *service.ThemeAutoUpdateReport
		err    error
	)
	if input.DryRun {
		report, err = h.autoUpdate.Check(c.Request.Context())
	} else {
		report, err = h.autoUpdate.Run(c.Request.Context(), true)
	}
	if err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, report)
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
