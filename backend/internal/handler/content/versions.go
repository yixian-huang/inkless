package content

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// VersionListItem is one row in the versions list.
type VersionListItem struct {
	Version    int    `json:"version"`
	ChangeNote string `json:"changeNote"`
	Operator   string `json:"operator"`
	CreatedAt  string `json:"createdAt"`
}

// GetVersionsResponse is GET .../versions.
type GetVersionsResponse struct {
	Items []VersionListItem `json:"items"`
	Total int64             `json:"total"`
}

// GetVersionDetailResponse is GET .../versions/{version}.
type GetVersionDetailResponse struct {
	ID          uint          `json:"id"`
	PageKey     string        `json:"pageKey"`
	Version     int           `json:"version"`
	Config      model.JSONMap `json:"config"`
	PublishedAt time.Time     `json:"publishedAt"`
	CreatedBy   uint          `json:"createdBy"`
}

// GetVersions lists published content versions.
// @Summary      List theme content versions
// @Tags         Content (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey  path  string true  "Page key"
// @Param        page     query int    false "Page number" default(1)
// @Param        pageSize query int    false "Page size" default(20)
// @Success      200 {object} GetVersionsResponse
// @Router       /admin/content/{pageKey}/versions [get]
func (h *Handler) GetVersions(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	page := 1
	pageSize := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	offset := (page - 1) * pageSize

	versions, total, err := h.versionRepo.ListByPageKey(c.Request.Context(), pageKey, offset, pageSize)
	if err != nil {
		apierror.Write(c, apierror.InternalServerError("Failed to fetch versions"))
		return
	}

	items := make([]VersionListItem, len(versions))
	for i, v := range versions {
		items[i] = VersionListItem{
			Version:   v.Version,
			CreatedAt: v.PublishedAt.Format(time.RFC3339),
		}
	}
	c.JSON(http.StatusOK, GetVersionsResponse{Items: items, Total: total})
}

// GetVersionDetail returns one published version snapshot.
// @Summary      Get theme content version detail
// @Tags         Content (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey path string true "Page key"
// @Param        version path int    true "Version number"
// @Success      200 {object} GetVersionDetailResponse
// @Failure      404 {object} object{error=object}
// @Router       /admin/content/{pageKey}/versions/{version} [get]
func (h *Handler) GetVersionDetail(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		apierror.Write(c, apierror.BadRequest("Invalid version parameter"))
		return
	}

	versionRecord, err := h.versionRepo.FindByPageKeyAndVersion(c.Request.Context(), pageKey, version)
	if err != nil {
		if isNotFoundErr(err) {
			apierror.Write(c, apierror.NotFound("Version not found"))
			return
		}
		apierror.Write(c, apierror.InternalServerError("Failed to fetch version detail"))
		return
	}

	c.JSON(http.StatusOK, GetVersionDetailResponse{
		ID:          versionRecord.ID,
		PageKey:     string(versionRecord.PageKey),
		Version:     versionRecord.Version,
		Config:      versionRecord.Config,
		PublishedAt: versionRecord.PublishedAt,
		CreatedBy:   versionRecord.CreatedBy,
	})
}
