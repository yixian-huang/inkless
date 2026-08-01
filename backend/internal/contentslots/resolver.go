package contentslots

import (
	"context"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// ThemeLookup is the subset of InstalledThemeRepository we need.
type ThemeLookup interface {
	FindActive(ctx context.Context) (*model.InstalledTheme, error)
}

// Resolver loads contentSlots for the active theme.
type Resolver struct {
	themes   ThemeLookup
	registry *Registry
}

// NewResolver builds a resolver with the given theme repo and registry.
func NewResolver(themes ThemeLookup, registry *Registry) *Resolver {
	if registry == nil {
		registry = DefaultRegistry()
	}
	return &Resolver{themes: themes, registry: registry}
}

// ResolveActive returns slots for the currently active theme.
func (r *Resolver) ResolveActive(ctx context.Context) ResolveResult {
	out := ResolveResult{Source: "none", Slots: nil}
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

	// 1) installed_themes.config.contentSlots (runtime override / future install)
	if m, ok := ManifestFromThemeConfig(themeID, version, config); ok {
		out.Slots = m.ContentSlots
		out.Source = "theme"
		if version != "" {
			out.ActiveThemeVersion = version
		}
		return out
	}

	// 2) host registry (built-in embeds)
	if r.registry != nil {
		if m, ok := r.registry.Get(themeID); ok && len(m.ContentSlots) > 0 {
			out.Slots = m.ContentSlots
			out.Source = "theme"
			if m.Version != "" && out.ActiveThemeVersion == "" {
				out.ActiveThemeVersion = m.Version
			}
			return out
		}
	}

	out.Source = "host-fallback"
	return out
}

// ResolveSlot returns the slot for pageKey under the active theme, if any.
func (r *Resolver) ResolveSlot(ctx context.Context, pageKey string) (ResolveResult, Slot, bool) {
	res := r.ResolveActive(ctx)
	for _, s := range res.Slots {
		if s.PageKey == pageKey {
			return res, s, true
		}
	}
	return res, Slot{}, false
}

// HostPageKeys is the host whitelist minus internal theme blob.
func HostPageKeys() []string {
	out := make([]string, 0, len(model.ValidPageKeys))
	for _, k := range model.ValidPageKeys {
		if k == model.PageKeyTheme {
			continue
		}
		out = append(out, string(k))
	}
	return out
}

// Ensure repository type is used (compile-time optional).
var _ ThemeLookup = (repository.InstalledThemeRepository)(nil)
