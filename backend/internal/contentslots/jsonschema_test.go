package contentslots

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONSchema_RejectsWrongInstallCodeType(t *testing.T) {
	slot := ProductFirstManifest().ContentSlots[0]
	cfg := map[string]any{
		"install": map[string]any{
			"code": map[string]any{"zh": "a", "en": "b"},
		},
	}
	errs := ValidateJSONSchema(cfg, slot.SchemaInline)
	require.NotEmpty(t, errs)
	assert.Equal(t, "SCHEMA", errs[0].Code)
}

func TestValidateJSONSchema_AllowsValidProductHome(t *testing.T) {
	slot := ProductFirstManifest().ContentSlots[0]
	cfg := map[string]any{
		"hero": map[string]any{
			"title": map[string]any{"zh": "产品", "en": "Product"},
			"media": map[string]any{"url": "/h.png", "alt": "H"},
		},
		"install": map[string]any{"code": "curl | sh"},
	}
	errs := ValidateJSONSchema(cfg, slot.SchemaInline)
	assert.Empty(t, errs, "%v", errs)
}

func TestValidateConfigAgainstSlot_IncludesSchema(t *testing.T) {
	slot := ProductFirstManifest().ContentSlots[0]
	cfg := map[string]any{
		"install": map[string]any{"code": 123},
	}
	errs := ValidateConfigAgainstSlot(cfg, slot)
	require.NotEmpty(t, errs)
	found := false
	for _, e := range errs {
		if e.Code == "SCHEMA" || e.Code == "TYPE" {
			found = true
		}
	}
	assert.True(t, found, "%v", errs)
}

func TestBlogFirstManifest_Registered(t *testing.T) {
	m, ok := DefaultRegistry().Get("blog-first")
	require.True(t, ok)
	require.NotEmpty(t, m.ContentSlots)
	assert.Equal(t, "blog-first/home@1", m.ContentSlots[0].SchemaID)
}
