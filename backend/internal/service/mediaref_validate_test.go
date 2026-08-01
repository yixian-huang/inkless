package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMediaRefLeafErrors_RejectsBilingualCaption(t *testing.T) {
	cfg := map[string]interface{}{
		"hero": map[string]interface{}{
			"title": map[string]interface{}{"zh": "标题", "en": "Title"},
			"media": map[string]interface{}{
				"url": "/uploads/a.png",
				"alt": "ok",
				"caption": map[string]interface{}{
					"zh": "说明",
					"en": "Caption",
				},
			},
		},
	}
	errs := CollectMediaRefLeafErrors(cfg)
	require.NotEmpty(t, errs)
	assert.Equal(t, "MEDIAREF_TYPE", errs[0].Code)
	assert.Contains(t, errs[0].Path, "caption")
}

func TestCollectMediaRefLeafErrors_AllowsStringLeaves(t *testing.T) {
	cfg := map[string]interface{}{
		"showcase": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"url":     "/a.png",
					"alt":     "A",
					"caption": "Cap",
				},
			},
		},
	}
	errs := CollectMediaRefLeafErrors(cfg)
	assert.Empty(t, errs)
}

func TestCollectMediaRefLeafErrors_AllowsLocalizedTextElsewhere(t *testing.T) {
	cfg := map[string]interface{}{
		"hero": map[string]interface{}{
			"title": map[string]interface{}{"zh": "中", "en": "En"},
			"subtitle": map[string]interface{}{"zh": "副", "en": "Sub"},
		},
	}
	errs := CollectMediaRefLeafErrors(cfg)
	assert.Empty(t, errs)
}
