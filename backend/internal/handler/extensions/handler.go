package extensions

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// Handler serves Phase A official extension-store endpoints (theme catalog first).
type Handler struct {
	loader      *themecatalog.Loader
	themeRepo   repository.InstalledThemeRepository
	hostVersion string
	allowHosts  []string
	builtinIDs  map[string]struct{}
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
