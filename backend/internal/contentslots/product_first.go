package contentslots

import "github.com/yixian-huang/inkless/backend/internal/builtinthemes"

// ProductFirstManifest is the host-embedded contract for product-first home.
// Keep in lockstep with inkless-theme-product-first/inkless.theme.json contentSlots.
func ProductFirstManifest() Manifest {
	return Manifest{
		ThemeID: builtinthemes.ProductFirst,
		Version: "0.1.9",
		ContentSlots: []Slot{
			{
				PageKey:  "home",
				SchemaID: "product-first/home@1",
				Title:    &LocalizedLabel{Zh: "产品首页", En: "Product home"},
				Description: "Landing: hero / showcase / features / howItWorks / install / bottomCta",
				SchemaPath:  "schemas/content/home.schema.json",
				MediaRefPaths: []string{
					"hero.media",
					"showcase.items[]",
					"features.items[].media",
				},
				LocalizedPaths: []string{
					"hero.eyebrow",
					"hero.title",
					"hero.subtitle",
					"hero.badge",
					"hero.primaryCta.label",
					"hero.secondaryCta.label",
					"showcase.title",
					"features.title",
					"features.items[].title",
					"features.items[].description",
					"howItWorks.title",
					"howItWorks.steps[].title",
					"howItWorks.steps[].description",
					"install.title",
					"install.caption",
					"bottomCta.title",
					"bottomCta.subtitle",
					"bottomCta.primaryCta.label",
				},
				StringPaths: []string{
					"install.code",
					"hero.primaryCta.href",
					"hero.secondaryCta.href",
					"bottomCta.primaryCta.href",
					"features.items[].href",
					"features.items[].icon",
				},
				SchemaInline: productFirstHomeJSONSchema(),
			},
		},
	}
}

func productFirstHomeJSONSchema() map[string]any {
	// Path rules remain primary; schema adds type checks for known sections.
	localized := map[string]any{
		"type": []any{"string", "object"},
	}
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
		"$id":                  "product-first/home@1",
		"type":                 "object",
		"additionalProperties": true,
		"properties": map[string]any{
			"hero": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    localized,
					"subtitle": localized,
					"eyebrow":  localized,
					"badge":    localized,
					"media":    mediaRef,
					"primaryCta": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"label": localized,
							"href":  map[string]any{"type": "string"},
						},
					},
					"secondaryCta": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"label": localized,
							"href":  map[string]any{"type": "string"},
						},
					},
				},
			},
			"showcase": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": localized,
					"items": map[string]any{"type": "array", "items": mediaRef},
				},
			},
			"features": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": localized,
					"items": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"title":       localized,
								"description": localized,
								"icon":        map[string]any{"type": "string"},
								"href":        map[string]any{"type": "string"},
								"media":       mediaRef,
							},
						},
					},
				},
			},
			"howItWorks": map[string]any{"type": "object"},
			"install": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":   localized,
					"caption": localized,
					"code":    map[string]any{"type": "string"},
				},
			},
			"bottomCta": map[string]any{"type": "object"},
		},
	}
}
