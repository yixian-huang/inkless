package themetemplates

import (
	"context"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

// ThemeLookup is the subset of InstalledThemeRepository we need.
type ThemeLookup interface {
	FindActive(ctx context.Context) (*model.InstalledTheme, error)
}

// Resolver loads templates for the active theme (native or contentSlots projection).
type Resolver struct {
	themes         ThemeLookup
	slotsRegistry  *contentslots.Registry
	// optional native template registry (future: parse templates[] from package)
	native *Registry
}

// Registry holds native templates[] embeds by theme id.
type Registry struct {
	byID map[string]Manifest
}

// NewRegistry creates an empty native template registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]Manifest)}
}

// Register stores a native templates manifest.
func (r *Registry) Register(m Manifest) {
	if r == nil || m.ThemeID == "" {
		return
	}
	if r.byID == nil {
		r.byID = make(map[string]Manifest)
	}
	cp := m
	if m.Templates != nil {
		cp.Templates = append([]Template(nil), m.Templates...)
	}
	if m.DefaultTemplates != nil {
		cp.DefaultTemplates = make(map[string]string, len(m.DefaultTemplates))
		for k, v := range m.DefaultTemplates {
			cp.DefaultTemplates[k] = v
		}
	}
	r.byID[m.ThemeID] = cp
}

// Get returns native manifest if any.
func (r *Registry) Get(themeID string) (Manifest, bool) {
	if r == nil {
		return Manifest{}, false
	}
	m, ok := r.byID[themeID]
	return m, ok
}

// DefaultNativeRegistry is empty for now; official themes use contentSlots projection.
func DefaultNativeRegistry() *Registry {
	return NewRegistry()
}

// NewResolver builds a templates resolver.
func NewResolver(themes ThemeLookup, slots *contentslots.Registry, native *Registry) *Resolver {
	if slots == nil {
		slots = contentslots.DefaultRegistry()
	}
	if native == nil {
		native = DefaultNativeRegistry()
	}
	return &Resolver{themes: themes, slotsRegistry: slots, native: native}
}

// ResolveActive returns templates for the active theme.
// Priority: installed_themes.config.templates → native registry → contentSlots projection.
func (r *Resolver) ResolveActive(ctx context.Context) ResolveResult {
	out := ResolveResult{Source: "none", DefaultTemplates: map[string]string{}}
	if r == nil {
		return out
	}
	var themeID, version string
	var config model.JSONMap
	if r.themes != nil {
		t, err := r.themes.FindActive(ctx)
		if err == nil && t != nil {
			themeID = t.ThemeID
			version = t.Version
			config = t.Config
			out.ActiveThemeID = themeID
			out.ActiveThemeVersion = version
		}
	}
	if themeID == "" {
		return out
	}

	// 1) config.templates JSON
	if m, ok := manifestFromConfig(themeID, version, config); ok {
		out.Templates = m.Templates
		out.DefaultTemplates = m.DefaultTemplates
		out.Source = "theme"
		return out
	}

	// 2) native host embeds
	if r.native != nil {
		if m, ok := r.native.Get(themeID); ok && len(m.Templates) > 0 {
			out.Templates = m.Templates
			out.DefaultTemplates = m.DefaultTemplates
			out.Source = "theme"
			return out
		}
	}

	// 3) project contentSlots (read-only v0 contract)
	if r.slotsRegistry != nil {
		if sm, ok := r.slotsRegistry.Get(themeID); ok && len(sm.ContentSlots) > 0 {
			m := ProjectManifest(sm)
			out.Templates = m.Templates
			out.DefaultTemplates = m.DefaultTemplates
			out.Source = "projection"
			return out
		}
		// also try config contentSlots via contentslots helper
		if sm, ok := contentslots.ManifestFromThemeConfig(themeID, version, config); ok {
			m := ProjectManifest(sm)
			out.Templates = m.Templates
			out.DefaultTemplates = m.DefaultTemplates
			out.Source = "projection"
			return out
		}
	}

	out.Source = "none"
	return out
}

// ResolveTemplate returns one template by key for the active theme.
func (r *Resolver) ResolveTemplate(ctx context.Context, key string) (ResolveResult, Template, bool) {
	res := r.ResolveActive(ctx)
	t, ok := FindByKey(res.Templates, key)
	return res, t, ok
}

func manifestFromConfig(themeID, version string, config map[string]any) (Manifest, bool) {
	if config == nil {
		return Manifest{}, false
	}
	raw, ok := config["templates"]
	if !ok || raw == nil {
		return Manifest{}, false
	}
	// Reuse JSON round-trip via contentslots-style marshal
	b, err := jsonMarshal(raw)
	if err != nil {
		return Manifest{}, false
	}
	var templates []Template
	if err := jsonUnmarshal(b, &templates); err != nil || len(templates) == 0 {
		return Manifest{}, false
	}
	m := Manifest{ThemeID: themeID, Version: version, Templates: templates}
	if d, ok := config["defaultTemplates"]; ok && d != nil {
		bb, _ := jsonMarshal(d)
		_ = jsonUnmarshal(bb, &m.DefaultTemplates)
	}
	return m, true
}
