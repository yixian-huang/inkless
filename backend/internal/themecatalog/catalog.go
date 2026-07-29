package themecatalog

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Catalog is the official themes index (schema v1).
type Catalog struct {
	SchemaVersion int          `json:"schemaVersion"`
	UpdatedAt     string       `json:"updatedAt"`
	Themes        []ThemeEntry `json:"themes"`
}

// ThemeEntry is one catalog row.
type ThemeEntry struct {
	Slug                string            `json:"slug"`
	ThemeID             string            `json:"themeId"`
	Name                string            `json:"name"`
	NameZh              string            `json:"nameZh"`
	Description         string            `json:"description"`
	DescriptionZh       string            `json:"descriptionZh"`
	Author              string            `json:"author"`
	Category            string            `json:"category"`
	Tags                []string          `json:"tags"`
	IconURL             string            `json:"iconUrl"`
	PreviewURL          string            `json:"previewUrl"`
	RepoURL             string            `json:"repoUrl"`
	ContractVersion     string            `json:"contractVersion"`
	MinHostVersion      string            `json:"minHostVersion"`
	Latest              ThemeVersion      `json:"latest"`
	Versions            []ThemeVersion    `json:"versions"`
	DefaultFeaturesHint map[string]any    `json:"defaultFeaturesHint,omitempty"`
	// BuiltinOnly marks host-bundled themes that activate without UMD download.
	BuiltinOnly bool `json:"builtinOnly,omitempty"`
	Official    bool `json:"official"`
}

// ThemeVersion is a downloadable (or metadata-only) release line.
type ThemeVersion struct {
	Version     string `json:"version"`
	UMDURL      string `json:"umdUrl"`
	Changelog   string `json:"changelog"`
	SHA256      string `json:"sha256"`
	PublishedAt string `json:"publishedAt"`
}

// Source labels where a catalog payload came from.
type Source string

const (
	SourceEmbedded Source = "embedded"
	SourceRemote   Source = "remote"
	SourceCache    Source = "cache"
)

// ParseCatalog unmarshals and validates a catalog JSON document.
func ParseCatalog(raw []byte) (*Catalog, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty catalog json")
	}
	var cat Catalog
	if err := json.Unmarshal(raw, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	if err := ValidateCatalog(&cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

// LoadEmbedded returns the compile-time fallback catalog.
func LoadEmbedded() (*Catalog, error) {
	return ParseCatalog(OfficialThemesJSON)
}

// ValidateCatalog checks schema version and required fields for official entries.
func ValidateCatalog(cat *Catalog) error {
	if cat == nil {
		return fmt.Errorf("catalog is nil")
	}
	if cat.SchemaVersion != 1 {
		return fmt.Errorf("unsupported catalog schemaVersion %d (want 1)", cat.SchemaVersion)
	}
	if len(cat.Themes) == 0 {
		return fmt.Errorf("catalog has no themes")
	}
	seenSlug := make(map[string]struct{}, len(cat.Themes))
	seenID := make(map[string]struct{}, len(cat.Themes))
	for i := range cat.Themes {
		t := &cat.Themes[i]
		if err := validateEntry(t); err != nil {
			return fmt.Errorf("themes[%d] %q: %w", i, t.Slug, err)
		}
		slug := strings.ToLower(strings.TrimSpace(t.Slug))
		id := strings.ToLower(strings.TrimSpace(t.ThemeID))
		if _, ok := seenSlug[slug]; ok {
			return fmt.Errorf("duplicate slug %q", t.Slug)
		}
		if _, ok := seenID[id]; ok {
			return fmt.Errorf("duplicate themeId %q", t.ThemeID)
		}
		seenSlug[slug] = struct{}{}
		seenID[id] = struct{}{}
	}
	return nil
}

func validateEntry(t *ThemeEntry) error {
	if t == nil {
		return fmt.Errorf("nil entry")
	}
	if strings.TrimSpace(t.Slug) == "" {
		return fmt.Errorf("slug is required")
	}
	if strings.TrimSpace(t.ThemeID) == "" {
		return fmt.Errorf("themeId is required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(t.ContractVersion) == "" {
		return fmt.Errorf("contractVersion is required")
	}
	if strings.TrimSpace(t.Latest.Version) == "" {
		return fmt.Errorf("latest.version is required")
	}
	// Builtin-only rows may omit umdUrl; marketplace installables need HTTPS UMD.
	if !t.BuiltinOnly {
		if strings.TrimSpace(t.Latest.UMDURL) == "" {
			return fmt.Errorf("latest.umdUrl is required unless builtinOnly")
		}
	}
	return nil
}

// FindBySlug returns a theme entry (case-insensitive slug).
func (c *Catalog) FindBySlug(slug string) (*ThemeEntry, bool) {
	if c == nil {
		return nil, false
	}
	want := strings.ToLower(strings.TrimSpace(slug))
	for i := range c.Themes {
		if strings.ToLower(strings.TrimSpace(c.Themes[i].Slug)) == want {
			return &c.Themes[i], true
		}
	}
	return nil, false
}

// ResolveVersion picks latest or a listed version for install.
func (t *ThemeEntry) ResolveVersion(version string) (*ThemeVersion, error) {
	if t == nil {
		return nil, fmt.Errorf("nil theme entry")
	}
	v := strings.TrimSpace(version)
	if v == "" || v == "latest" {
		cp := t.Latest
		return &cp, nil
	}
	for i := range t.Versions {
		if t.Versions[i].Version == v {
			cp := t.Versions[i]
			return &cp, nil
		}
	}
	// Allow matching latest even if versions[] omitted it.
	if t.Latest.Version == v {
		cp := t.Latest
		return &cp, nil
	}
	return nil, fmt.Errorf("version %q not found for theme %q", version, t.Slug)
}
