package themetemplates

import (
	"fmt"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
)

// ProjectSlot converts a legacy contentSlots entry into a page Template.
// contentSlots remain read-only discovery; templates are the target contract.
func ProjectSlot(themeID string, slot contentslots.Slot) Template {
	key := slot.SchemaID
	if key == "" {
		key = fmt.Sprintf("%s/%s", themeID, slot.PageKey)
	}
	// Prefer canonical template key themeId/pageKey when schemaId is product-first/home@1 style
	if slot.PageKey != "" && themeID != "" {
		// keep schemaId as-is for identity; also expose route via pageKey
	}
	title := slot.Title
	return Template{
		Key:            key,
		AppliesTo:      "page",
		Title:          title,
		Description:    slot.Description,
		SchemaPath:     slot.SchemaPath,
		SchemaInline:   slot.SchemaInline,
		MediaRefPaths:  append([]string(nil), slot.MediaRefPaths...),
		LocalizedPaths: append([]string(nil), slot.LocalizedPaths...),
		StringPaths:    append([]string(nil), slot.StringPaths...),
		Renderer:       "theme-page",
		RouteHint: &RouteHint{
			Slug: slot.PageKey,
			Path: pathForSlug(slot.PageKey),
			Nav:  slot.PageKey == "home" || slot.PageKey == "features",
		},
		Source: "contentSlots-projection",
	}
}

// ProjectManifest maps contentSlots manifest → templates manifest.
func ProjectManifest(m contentslots.Manifest) Manifest {
	out := Manifest{
		ThemeID: m.ThemeID,
		Version: m.Version,
		DefaultTemplates: map[string]string{},
	}
	for _, s := range m.ContentSlots {
		t := ProjectSlot(m.ThemeID, s)
		out.Templates = append(out.Templates, t)
		if s.PageKey == "home" {
			out.DefaultTemplates["home"] = t.Key
			out.DefaultTemplates["page"] = t.Key
		}
	}
	// default post chrome key for official themes
	if m.ThemeID != "" {
		postKey := m.ThemeID + "/post"
		out.Templates = append(out.Templates, Template{
			Key:       postKey,
			AppliesTo: "post",
			Title:     &Localized{Zh: "文章", En: "Post"},
			Renderer:  "theme-post",
			Source:    "host-default",
			Description: "Single post chrome; body remains article fields",
		})
		out.DefaultTemplates["post"] = postKey
	}
	return out
}

func pathForSlug(slug string) string {
	if slug == "" || slug == "home" {
		return "/"
	}
	return "/" + slug
}

// Summarize builds list DTOs without schema bodies.
func Summarize(templates []Template) []TemplateSummary {
	out := make([]TemplateSummary, 0, len(templates))
	for _, t := range templates {
		slug := ""
		if t.RouteHint != nil {
			slug = t.RouteHint.Slug
		}
		out = append(out, TemplateSummary{
			Key:         t.Key,
			AppliesTo:   t.AppliesTo,
			Title:       t.Title,
			Description: t.Description,
			Slug:        slug,
			Renderer:    t.Renderer,
			HasSchema:   t.SchemaInline != nil || t.SchemaPath != "",
			Source:      t.Source,
		})
	}
	return out
}

// FindByKey returns a template by key.
func FindByKey(templates []Template, key string) (Template, bool) {
	for _, t := range templates {
		if t.Key == key {
			return t, true
		}
	}
	return Template{}, false
}

// FindBySlug returns the first page template whose routeHint.slug matches.
func FindBySlug(templates []Template, slug string) (Template, bool) {
	for _, t := range templates {
		if t.AppliesTo != "page" {
			continue
		}
		if t.RouteHint != nil && t.RouteHint.Slug == slug {
			return t, true
		}
	}
	return Template{}, false
}
