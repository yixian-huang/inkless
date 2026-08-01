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
	// Lightweight draft-07-ish object schema for discovery + future validator.
	// Path rules (mediaRef/string/localized) are the enforcement MVP.
	return map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id":     "product-first/home@1",
		"type":    "object",
		"additionalProperties": true,
		"properties": map[string]any{
			"hero":        map[string]any{"type": "object"},
			"showcase":    map[string]any{"type": "object"},
			"features":    map[string]any{"type": "object"},
			"howItWorks":  map[string]any{"type": "object"},
			"install":     map[string]any{"type": "object"},
			"bottomCta":   map[string]any{"type": "object"},
		},
	}
}
