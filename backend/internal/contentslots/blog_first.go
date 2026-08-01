package contentslots

import "github.com/yixian-huang/inkless/backend/internal/builtinthemes"

// BlogFirstManifest declares optional home content overrides.
// blog-first home primarily uses site config + articles; content_documents.home
// may hold optional intro/cover overrides for agents without requiring fields.
func BlogFirstManifest() Manifest {
	return Manifest{
		ThemeID: builtinthemes.BlogFirst,
		Version: "1.0.0",
		ContentSlots: []Slot{
			{
				PageKey:     "home",
				SchemaID:    "blog-first/home@1",
				Title:       &LocalizedLabel{Zh: "博客首页", En: "Blog home"},
				Description: "Optional home overrides; author identity lives in site config; posts from articles API",
				SchemaPath:  "schemas/content/home.schema.json",
				MediaRefPaths: []string{
					"cover",
					"hero.media",
				},
				LocalizedPaths: []string{
					"title",
					"intro",
					"recentHeading",
					"hero.title",
					"hero.subtitle",
				},
				StringPaths: []string{
					"moreHref",
					"moreLabel",
				},
				SchemaInline: blogFirstHomeJSONSchema(),
			},
		},
	}
}

func blogFirstHomeJSONSchema() map[string]any {
	localized := map[string]any{"type": []any{"string", "object"}}
	mediaRef := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":     map[string]any{"type": "string"},
			"alt":     map[string]any{"type": "string"},
			"caption": map[string]any{"type": "string"},
		},
		"additionalProperties": true,
	}
	return map[string]any{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"$id":                  "blog-first/home@1",
		"type":                 "object",
		"additionalProperties": true,
		"properties": map[string]any{
			"title":         localized,
			"intro":         localized,
			"recentHeading": localized,
			"moreHref":      map[string]any{"type": "string"},
			"moreLabel":     map[string]any{"type": "string"},
			"cover":         mediaRef,
			"hero": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    localized,
					"subtitle": localized,
					"media":    mediaRef,
				},
			},
		},
	}
}
