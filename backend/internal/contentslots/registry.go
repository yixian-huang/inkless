package contentslots

import (
	"encoding/json"
	"sync"
)

// Registry holds built-in / known theme content contracts by theme id.
type Registry struct {
	mu    sync.RWMutex
	byID  map[string]Manifest
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]Manifest)}
}

// DefaultRegistry returns a registry preloaded with official embeds.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(ProductFirstManifest())
	r.Register(BlogFirstManifest())
	return r
}

// Register stores or replaces a manifest for themeID.
func (r *Registry) Register(m Manifest) {
	if r == nil || m.ThemeID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byID == nil {
		r.byID = make(map[string]Manifest)
	}
	// copy slots slice
	cp := m
	if m.ContentSlots != nil {
		cp.ContentSlots = append([]Slot(nil), m.ContentSlots...)
	}
	r.byID[m.ThemeID] = cp
}

// Get returns a manifest by theme id.
func (r *Registry) Get(themeID string) (Manifest, bool) {
	if r == nil {
		return Manifest{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.byID[themeID]
	return m, ok
}

// ParseManifestJSON parses inkless.theme.json content (full or partial with contentSlots).
func ParseManifestJSON(themeID string, raw []byte) (Manifest, error) {
	var envelope struct {
		ID           string `json:"id"`
		Version      string `json:"version"`
		ContentSlots []Slot `json:"contentSlots"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Manifest{}, err
	}
	id := envelope.ID
	if id == "" {
		id = themeID
	}
	return Manifest{
		ThemeID:      id,
		Version:      envelope.Version,
		ContentSlots: envelope.ContentSlots,
	}, nil
}

// ManifestFromThemeConfig reads contentSlots from installed_themes.config JSONMap.
func ManifestFromThemeConfig(themeID, version string, config map[string]any) (Manifest, bool) {
	if config == nil {
		return Manifest{}, false
	}
	rawSlots, ok := config["contentSlots"]
	if !ok || rawSlots == nil {
		return Manifest{}, false
	}
	b, err := json.Marshal(rawSlots)
	if err != nil {
		return Manifest{}, false
	}
	var slots []Slot
	if err := json.Unmarshal(b, &slots); err != nil || len(slots) == 0 {
		return Manifest{}, false
	}
	return Manifest{
		ThemeID:      themeID,
		Version:      version,
		ContentSlots: slots,
	}, true
}

// FindSlot returns the slot for pageKey in a manifest.
func FindSlot(m Manifest, pageKey string) (Slot, bool) {
	for _, s := range m.ContentSlots {
		if s.PageKey == pageKey {
			return s, true
		}
	}
	return Slot{}, false
}
