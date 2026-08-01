package content

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// ListSlotsResponse is GET /admin/content/slots.
type ListSlotsResponse struct {
	ActiveThemeID      string                    `json:"activeThemeId,omitempty"`
	ActiveThemeVersion string                    `json:"activeThemeVersion,omitempty"`
	Source             string                    `json:"source"`
	Slots              []contentslots.SlotSummary `json:"slots"`
	HostPageKeys       []string                  `json:"hostPageKeys"`
}

// ListSlots returns theme content slots for the active theme.
// @Summary      List theme content slots
// @Description  Discovery: active theme contentSlots (no full JSON Schema body)
// @Tags         Content (Admin)
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} ListSlotsResponse
// @Router       /admin/content/slots [get]
func (h *Handler) ListSlots(c *gin.Context) {
	res := contentslots.ResolveResult{Source: "none"}
	if h.slots != nil {
		res = h.slots.ResolveActive(c.Request.Context())
	}
	summaries := make([]contentslots.SlotSummary, 0, len(res.Slots))
	for _, s := range res.Slots {
		summaries = append(summaries, contentslots.SlotSummary{
			PageKey:     s.PageKey,
			SchemaID:    s.SchemaID,
			Title:       s.Title,
			Description: s.Description,
			HasSchema:   s.SchemaInline != nil || s.SchemaPath != "",
		})
	}
	c.JSON(http.StatusOK, ListSlotsResponse{
		ActiveThemeID:      res.ActiveThemeID,
		ActiveThemeVersion: res.ActiveThemeVersion,
		Source:             res.Source,
		Slots:              summaries,
		HostPageKeys:       contentslots.HostPageKeys(),
	})
}

// GetSchema returns the content contract for a pageKey under the active theme.
// @Summary      Get theme content schema for pageKey
// @Tags         Content (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        pageKey path string true "Page key"
// @Success      200 {object} contentslots.SchemaPayload
// @Failure      400 {object} object{error=object}
// @Router       /admin/content/{pageKey}/schema [get]
func (h *Handler) GetSchema(c *gin.Context) {
	pageKey := model.PageKey(c.Param("pageKey"))
	if !isValidPageKey(pageKey) {
		apierror.Write(c, apierror.BadRequest("Invalid page key"))
		return
	}

	payload := contentslots.SchemaPayload{
		PageKey: string(pageKey),
		Source:  "host-fallback",
	}
	if h.slots != nil {
		res, slot, ok := h.slots.ResolveSlot(c.Request.Context(), string(pageKey))
		payload.ActiveThemeID = res.ActiveThemeID
		payload.ActiveThemeVersion = res.ActiveThemeVersion
		if ok {
			payload.Source = "theme"
			payload.SchemaID = slot.SchemaID
			payload.MediaRefPaths = slot.MediaRefPaths
			payload.LocalizedPaths = slot.LocalizedPaths
			payload.StringPaths = slot.StringPaths
			payload.JSONSchema = slot.SchemaInline
			payload.Description = slot.Description
		} else if res.Source != "" {
			payload.Source = res.Source
			if res.Source == "none" {
				payload.Source = "host-fallback"
			}
		}
	}
	c.JSON(http.StatusOK, payload)
}
