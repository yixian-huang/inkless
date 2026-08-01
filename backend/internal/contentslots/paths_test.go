package contentslots

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfigAgainstSlot_MediaAndString(t *testing.T) {
	slot := ProductFirstManifest().ContentSlots[0]
	cfg := map[string]any{
		"hero": map[string]any{
			"title": map[string]any{"zh": "产品", "en": "Product"},
			"media": map[string]any{
				"url": "/h.png",
				"caption": map[string]any{"zh": "中", "en": "En"},
			},
			"primaryCta": map[string]any{"href": "/start", "label": map[string]any{"zh": "开始", "en": "Start"}},
		},
		"install": map[string]any{
			"code": map[string]any{"zh": "a", "en": "b"},
		},
		"showcase": map[string]any{
			"items": []any{
				map[string]any{"url": "/a.png", "alt": "A"},
			},
		},
	}
	errs := ValidateConfigAgainstSlot(cfg, slot)
	require.NotEmpty(t, errs)
	var codes []string
	for _, e := range errs {
		codes = append(codes, e.Code+":"+e.Path)
	}
	assert.Contains(t, codes, "MEDIAREF_TYPE:hero.media.caption")
	assert.Contains(t, codes, "TYPE:install.code")
}

func TestValidateConfigAgainstSlot_OK(t *testing.T) {
	slot := ProductFirstManifest().ContentSlots[0]
	cfg := map[string]any{
		"hero": map[string]any{
			"title": map[string]any{"zh": "产品", "en": "Product"},
			"media": map[string]any{"url": "/h.png", "alt": "H", "caption": "c"},
			"primaryCta": map[string]any{
				"href":  "/x",
				"label": map[string]any{"zh": "去", "en": "Go"},
			},
		},
		"install": map[string]any{"code": "curl | sh"},
		"showcase": map[string]any{
			"items": []any{
				map[string]any{"url": "/a.png", "alt": "A"},
			},
		},
		"features": map[string]any{
			"items": []any{
				map[string]any{
					"title": map[string]any{"zh": "快", "en": "Fast"},
					"media": map[string]any{"url": "/f.png"},
				},
			},
		},
	}
	errs := ValidateConfigAgainstSlot(cfg, slot)
	assert.Empty(t, errs, "%v", errs)
}

func TestResolvePath_Array(t *testing.T) {
	cfg := map[string]any{
		"showcase": map[string]any{
			"items": []any{
				map[string]any{"url": "/1.png"},
				map[string]any{"url": "/2.png"},
			},
		},
	}
	nodes := resolvePath(cfg, "showcase.items[]")
	require.Len(t, nodes, 2)
}
