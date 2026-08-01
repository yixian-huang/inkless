package themetemplates

import "github.com/yixian-huang/inkless/backend/internal/contentslots"

// Localized is a short bilingual label.
type Localized = contentslots.LocalizedLabel

// RouteHint suggests seed slug/path/nav for a page template.
type RouteHint struct {
	Slug      string `json:"slug,omitempty"`
	Path      string `json:"path,omitempty"`
	Nav       bool   `json:"nav,omitempty"`
	SortOrder int    `json:"sortOrder,omitempty"`
}

// Template is a theme display contract for Page or Post (theme-as-templates T4).
type Template struct {
	Key            string         `json:"key"`
	AppliesTo      string         `json:"appliesTo"` // page | post
	Title          *Localized     `json:"title,omitempty"`
	Description    string         `json:"description,omitempty"`
	RouteHint      *RouteHint     `json:"routeHint,omitempty"`
	SchemaPath     string         `json:"schema,omitempty"`
	DefaultSeed    string         `json:"defaultSeed,omitempty"`
	Renderer       string         `json:"renderer,omitempty"` // theme-page | composable | theme-post
	SectionTypes   []string       `json:"sectionTypes,omitempty"`
	MediaRefPaths  []string       `json:"mediaRefPaths,omitempty"`
	LocalizedPaths []string       `json:"localizedPaths,omitempty"`
	StringPaths    []string       `json:"stringPaths,omitempty"`
	SchemaInline   map[string]any `json:"schemaInline,omitempty"`
	// Source explains origin: templates | contentSlots-projection
	Source string `json:"source,omitempty"`
}

// TemplateSummary is a token-light list entry.
type TemplateSummary struct {
	Key         string     `json:"key"`
	AppliesTo   string     `json:"appliesTo"`
	Title       *Localized `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Slug        string     `json:"slug,omitempty"`
	Renderer    string     `json:"renderer,omitempty"`
	HasSchema   bool       `json:"hasSchema"`
	Source      string     `json:"source,omitempty"`
}

// Manifest is theme templates[] + defaults.
type Manifest struct {
	ThemeID          string            `json:"id"`
	Version          string            `json:"version,omitempty"`
	Templates        []Template        `json:"templates"`
	DefaultTemplates map[string]string `json:"defaultTemplates,omitempty"` // page|post|home → key
}

// ResolveResult is active theme templates discovery.
type ResolveResult struct {
	ActiveThemeID      string
	ActiveThemeVersion string
	Source             string // theme | projection | none
	Templates          []Template
	DefaultTemplates   map[string]string
}
