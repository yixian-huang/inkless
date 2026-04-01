package qa

import (
	"testing"

	"blotting-consultancy/internal/model"

	"github.com/stretchr/testify/assert"
)

func TestExtractTextFromConfig(t *testing.T) {
	config := model.JSONMap{
		"title": "Welcome",
		"hero": map[string]interface{}{
			"heading":    "Our Company",
			"subheading": "Best consultancy",
		},
		"items": []interface{}{
			map[string]interface{}{"name": "Service A"},
			map[string]interface{}{"name": "Service B"},
		},
		"count": 42, // non-string values should be skipped
	}

	text := extractTextFromConfig(config)
	assert.Contains(t, text, "Welcome")
	assert.Contains(t, text, "Our Company")
	assert.Contains(t, text, "Best consultancy")
	assert.Contains(t, text, "Service A")
	assert.Contains(t, text, "Service B")
}

func TestExtractTextFromConfig_Nil(t *testing.T) {
	text := extractTextFromConfig(nil)
	assert.Equal(t, "", text)
}

func TestExtractTextFromConfig_Empty(t *testing.T) {
	text := extractTextFromConfig(model.JSONMap{})
	assert.Equal(t, "", text)
}

func TestJoinNonEmpty(t *testing.T) {
	assert.Equal(t, "", joinNonEmpty(nil, "\n"))
	assert.Equal(t, "", joinNonEmpty([]string{}, "\n"))
	assert.Equal(t, "a", joinNonEmpty([]string{"a"}, "\n"))
	assert.Equal(t, "a\nb", joinNonEmpty([]string{"a", "", "b"}, "\n"))
}
