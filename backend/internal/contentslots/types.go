package contentslots

// LocalizedLabel is a short bilingual label for agent/admin UI.
type LocalizedLabel struct {
	Zh string `json:"zh,omitempty"`
	En string `json:"en,omitempty"`
}

// Slot is one theme-bound content_documents pageKey contract.
type Slot struct {
	PageKey         string            `json:"pageKey"`
	SchemaID        string            `json:"schemaId"`
	Title           *LocalizedLabel   `json:"title,omitempty"`
	Description     string            `json:"description,omitempty"`
	SchemaPath      string            `json:"schema,omitempty"`      // relative path in theme package
	SchemaInline    map[string]any    `json:"schemaInline,omitempty"` // optional embedded JSON Schema
	MediaRefPaths   []string          `json:"mediaRefPaths,omitempty"`
	LocalizedPaths  []string          `json:"localizedPaths,omitempty"`
	StringPaths     []string          `json:"stringPaths,omitempty"`
}

// Manifest is the contentSlots section of inkless.theme.json (or host embed).
type Manifest struct {
	ThemeID      string `json:"id"`
	Version      string `json:"version,omitempty"`
	ContentSlots []Slot `json:"contentSlots"`
}

// SlotSummary is a token-light listing entry.
type SlotSummary struct {
	PageKey     string          `json:"pageKey"`
	SchemaID    string          `json:"schemaId"`
	Title       *LocalizedLabel `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	HasSchema   bool            `json:"hasSchema"`
}

// ResolveResult is the active theme + slots for discovery APIs.
type ResolveResult struct {
	ActiveThemeID      string
	ActiveThemeVersion string
	Source             string // theme | host-fallback | none
	Slots              []Slot
}

// SchemaPayload is GET /admin/content/:pageKey/schema body.
type SchemaPayload struct {
	PageKey         string         `json:"pageKey"`
	ActiveThemeID   string         `json:"activeThemeId,omitempty"`
	ActiveThemeVersion string      `json:"activeThemeVersion,omitempty"`
	SchemaID        string         `json:"schemaId,omitempty"`
	MediaRefPaths   []string       `json:"mediaRefPaths,omitempty"`
	LocalizedPaths  []string       `json:"localizedPaths,omitempty"`
	StringPaths     []string       `json:"stringPaths,omitempty"`
	JSONSchema      map[string]any `json:"jsonSchema,omitempty"`
	Source          string         `json:"source"` // theme | host-fallback
	Description     string         `json:"description,omitempty"`
}
