package service

import (
	"crypto/rand"
	"fmt"
	"strings"

	"blotting-consultancy/internal/model"
)

// sectionMapping describes how to create a section from a content doc key.
type sectionMapping struct {
	SectionType string
	Variant     string
	ConfigKey   string // top-level key in the content doc config
}

// pageKeyMappings maps each page key to its ordered section definitions.
var pageKeyMappings = map[string][]sectionMapping{
	"home": {
		{SectionType: "hero", Variant: "fullscreen", ConfigKey: "hero"},
		{SectionType: "card-grid", Variant: "three-column", ConfigKey: "advantages"},
		{SectionType: "company-profile", Variant: "default", ConfigKey: "about"},
		{SectionType: "service-cards", Variant: "default", ConfigKey: "coreServices"},
		{SectionType: "team-grid", Variant: "default", ConfigKey: "team"},
		{SectionType: "contact-form", Variant: "default", ConfigKey: "contact"},
	},
	"about": {
		{SectionType: "hero", Variant: "default", ConfigKey: "hero"},
		{SectionType: "company-profile", Variant: "default", ConfigKey: "companyProfile"},
		{SectionType: "rich-text", Variant: "default", ConfigKey: "blocks"},
		{SectionType: "team-grid", Variant: "default", ConfigKey: "team"},
	},
	"advantages": {
		{SectionType: "hero", Variant: "default", ConfigKey: "hero"},
		{SectionType: "checklist", Variant: "default", ConfigKey: "blocks"},
	},
	"core-services": {
		{SectionType: "hero", Variant: "default", ConfigKey: "hero"},
		{SectionType: "service-cards", Variant: "default", ConfigKey: "services"},
		{SectionType: "rich-text", Variant: "default", ConfigKey: "details"},
	},
	"cases": {
		{SectionType: "hero", Variant: "default", ConfigKey: "hero"},
		{SectionType: "card-grid", Variant: "default", ConfigKey: "cases"},
	},
	"experts": {
		{SectionType: "hero", Variant: "default", ConfigKey: "hero"},
		{SectionType: "team-grid", Variant: "default", ConfigKey: "experts"},
	},
	"contact": {
		{SectionType: "hero", Variant: "default", ConfigKey: "hero"},
		{SectionType: "contact-form", Variant: "default", ConfigKey: "contactInfo"},
		{SectionType: "text-image", Variant: "default", ConfigKey: "map"},
	},
}

// localizableFieldsBySection maps section types to their localizable field paths.
var localizableFieldsBySection = map[string][]string{
	"hero":            {"title", "subtitle", "cta.text"},
	"card-grid":       {"title", "subtitle", "cards[].title", "cards[].description"},
	"rich-text":       {"content", "title"},
	"contact-form":    {"title", "subtitle", "submitText", "fields[].label", "fields[].placeholder"},
	"service-cards":   {"title", "subtitle", "cards[].title", "cards[].description"},
	"team-grid":       {"title", "subtitle", "members[].name", "members[].role", "members[].bio"},
	"text-image":      {"title", "text"},
	"checklist":       {"title", "items[].title", "items[].description"},
	"company-profile": {"title", "description", "stats[].label"},
}

// generateID produces a short random hex ID for sections.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ConvertContentDocToSections transforms a content document's object-keyed config
// into a {sections: [...]} array format using predefined section mappings per page key.
func ConvertContentDocToSections(pageKey string, config model.JSONMap) model.JSONMap {
	mappings, ok := pageKeyMappings[pageKey]
	if !ok {
		// Unknown page key — wrap entire config as a single rich-text section
		return model.JSONMap{
			"sections": []interface{}{
				model.JSONMap{
					"id":      generateID(),
					"type":    "rich-text",
					"variant": "default",
					"locked":  false,
					"props":   config,
				},
			},
		}
	}

	sections := make([]interface{}, 0, len(mappings))
	for _, m := range mappings {
		props := model.JSONMap{}
		if val, exists := config[m.ConfigKey]; exists {
			if mapVal, ok := val.(map[string]interface{}); ok {
				props = model.JSONMap(mapVal)
			}
		}

		section := model.JSONMap{
			"id":      generateID(),
			"type":    m.SectionType,
			"variant": m.Variant,
			"locked":  false,
			"props":   props,
		}
		sections = append(sections, section)
	}

	return model.JSONMap{
		"sections": sections,
	}
}

// ConvertBlockPageToUnified adds variant/locked fields to existing sections
// and wraps localizable string fields in {zh, en} objects.
func ConvertBlockPageToUnified(config model.JSONMap) model.JSONMap {
	sectionsRaw, ok := config["sections"]
	if !ok {
		return config
	}

	sectionsSlice, ok := sectionsRaw.([]interface{})
	if !ok {
		return config
	}

	newSections := make([]interface{}, 0, len(sectionsSlice))
	for _, s := range sectionsSlice {
		sectionMap, ok := s.(map[string]interface{})
		if !ok {
			newSections = append(newSections, s)
			continue
		}

		// Ensure id
		if _, hasID := sectionMap["id"]; !hasID {
			sectionMap["id"] = generateID()
		}

		// Ensure variant
		if _, hasVariant := sectionMap["variant"]; !hasVariant {
			sectionMap["variant"] = "default"
		}

		// Ensure locked
		if _, hasLocked := sectionMap["locked"]; !hasLocked {
			sectionMap["locked"] = false
		}

		// Wrap localizable fields in props
		sectionType, _ := sectionMap["type"].(string)
		if paths, hasPaths := localizableFieldsBySection[sectionType]; hasPaths {
			if props, hasProps := sectionMap["props"].(map[string]interface{}); hasProps {
				for _, path := range paths {
					wrapPath(props, path)
				}
				sectionMap["props"] = props
			}
		}

		newSections = append(newSections, sectionMap)
	}

	return model.JSONMap{
		"sections": newSections,
	}
}

// wrapLocalizable wraps a plain string value into a bilingual object {zh: value, en: ""}.
// If the value is already a map (already bilingual), it is returned as-is.
func wrapLocalizable(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return map[string]interface{}{
			"zh": v,
			"en": "",
		}
	case map[string]interface{}:
		// Already bilingual or complex — leave it
		return v
	default:
		return value
	}
}

// wrapPath handles dot-notation paths (e.g., "cta.text") and array notation
// (e.g., "cards[].title") to wrap leaf string fields as bilingual objects.
func wrapPath(data map[string]interface{}, path string) {
	// Check for array notation
	if idx := strings.Index(path, "[]."); idx >= 0 {
		arrayKey := path[:idx]
		rest := path[idx+3:]

		arr, ok := data[arrayKey].([]interface{})
		if !ok {
			return
		}

		for _, item := range arr {
			if itemMap, ok := item.(map[string]interface{}); ok {
				wrapPath(itemMap, rest)
			}
		}
		return
	}

	// Check for dot notation
	if idx := strings.Index(path, "."); idx >= 0 {
		key := path[:idx]
		rest := path[idx+1:]

		nested, ok := data[key].(map[string]interface{})
		if !ok {
			return
		}

		wrapPath(nested, rest)
		return
	}

	// Leaf field — wrap it
	if val, exists := data[path]; exists {
		data[path] = wrapLocalizable(val)
	}
}

// unwrapPath extracts a value from a nested map using dot notation.
func unwrapPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}

	return current
}

// UnwrapLocalizable extracts a locale string from a bilingual object.
// If value is a plain string, returns it directly.
// If value is a map with "zh"/"en" keys, returns the requested locale.
func UnwrapLocalizable(value interface{}, locale string) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		if s, ok := v[locale].(string); ok {
			return s
		}
		// Fallback to zh
		if s, ok := v["zh"].(string); ok {
			return s
		}
		return ""
	default:
		return ""
	}
}
