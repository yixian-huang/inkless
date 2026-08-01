package agentcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepConfigDiff_NestedPaths(t *testing.T) {
	left := map[string]any{
		"hero": map[string]any{
			"title": map[string]any{"zh": "旧", "en": "Old"},
			"media": map[string]any{"url": "/a.png", "alt": "A"},
		},
		"features": map[string]any{
			"items": []any{
				map[string]any{"title": map[string]any{"zh": "一", "en": "One"}},
			},
		},
	}
	right := map[string]any{
		"hero": map[string]any{
			"title": map[string]any{"zh": "新", "en": "Old"},
			"media": map[string]any{"url": "/a.png", "alt": "A"},
		},
		"features": map[string]any{
			"items": []any{
				map[string]any{"title": map[string]any{"zh": "一", "en": "One"}},
				map[string]any{"title": map[string]any{"zh": "二", "en": "Two"}},
			},
		},
		"install": map[string]any{"code": "curl | sh"},
	}

	diff := DeepConfigDiff(left, right)
	sum := diff["summary"].(map[string]any)
	assert.GreaterOrEqual(t, sum["changed"].(int), 1)
	assert.GreaterOrEqual(t, sum["added"].(int), 1)

	paths, ok := diff["paths"].([]DiffOp)
	require.True(t, ok)
	var foundTitle, foundItem, foundInstall bool
	for _, p := range paths {
		if p.Path == "hero.title.zh" && p.Op == "changed" {
			foundTitle = true
			assert.Equal(t, "旧", p.From)
			assert.Equal(t, "新", p.To)
		}
		if p.Path == "features.items[1]" && p.Op == "added" {
			foundItem = true
		}
		if p.Path == "install" && p.Op == "added" {
			foundInstall = true
		}
	}
	assert.True(t, foundTitle, "paths=%v", paths)
	assert.True(t, foundItem, "paths=%v", paths)
	assert.True(t, foundInstall, "paths=%v", paths)
	assert.NotNil(t, diff["shallow"])
}

func TestLocalMediaRefIssues(t *testing.T) {
	cfg := map[string]any{
		"hero": map[string]any{
			"media": map[string]any{
				"url": "/x.png",
				"caption": map[string]any{"zh": "中", "en": "En"},
			},
		},
	}
	issues := LocalMediaRefIssues(cfg)
	require.NotEmpty(t, issues)
	assert.Equal(t, "MEDIAREF_TYPE", issues[0]["code"])
	assert.Contains(t, issues[0]["path"], "caption")

	ok := LocalMediaRefIssues(map[string]any{
		"hero": map[string]any{
			"title": map[string]any{"zh": "t", "en": "t"},
			"media": map[string]any{"url": "/x.png", "alt": "a"},
		},
	})
	assert.Empty(t, ok)
}
