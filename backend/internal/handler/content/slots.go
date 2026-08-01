package content

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/themetemplates"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

func projectTemplateSummaries(res themetemplates.ResolveResult) []themetemplates.TemplateSummary {
	return themetemplates.Summarize(res.Templates)
}

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
	// T4: also project contentSlots → templates (read-only discovery)
	var projected any
	if h.templates != nil {
		tres := h.templates.ResolveActive(c.Request.Context())
		projected = gin.H{
			"source":           tres.Source,
			"templates":        projectTemplateSummaries(tres),
			"defaultTemplates": tres.DefaultTemplates,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"activeThemeId":      res.ActiveThemeID,
		"activeThemeVersion": res.ActiveThemeVersion,
		"source":             res.Source,
		"slots":              summaries,
		"hostPageKeys":       contentslots.HostPageKeys(),
		"templatesProjection": projected,
		"deprecated":          "prefer GET /admin/themes/active/templates (theme-as-templates T4)",
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
	// T4: attach projected templateKey for agents moving to pages
	if h.templates != nil {
		tres := h.templates.ResolveActive(c.Request.Context())
		if t, ok := themetemplates.FindBySlug(tres.Templates, string(pageKey)); ok {
			c.Header("X-Inkless-Template-Key", t.Key)
			c.JSON(http.StatusOK, gin.H{
				"pageKey":            payload.PageKey,
				"activeThemeId":      payload.ActiveThemeID,
				"activeThemeVersion": payload.ActiveThemeVersion,
				"schemaId":           payload.SchemaID,
				"mediaRefPaths":      payload.MediaRefPaths,
				"localizedPaths":     payload.LocalizedPaths,
				"stringPaths":        payload.StringPaths,
				"jsonSchema":         payload.JSONSchema,
				"source":             payload.Source,
				"description":        payload.Description,
				"templateKey":        t.Key,
				"templateSource":     t.Source,
				"deprecated":         "prefer GET /admin/themes/active/template?key=…",
			})
			return
		}
	}
	c.JSON(http.StatusOK, payload)
}
