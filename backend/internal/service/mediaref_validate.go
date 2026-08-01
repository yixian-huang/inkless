package service

import (
	"fmt"
	"strings"
)

// mediaLeafKeys are fields on a MediaRef (and similar) that must be plain strings.
var mediaLeafKeys = map[string]struct{}{
	"url":     {},
	"alt":     {},
	"caption": {},
}

// mediaSlotKeys are parent object keys commonly holding a MediaRef.
var mediaSlotKeys = map[string]struct{}{
	"media":           {},
	"backgroundImage": {},
	"image":           {},
}

// CollectMediaRefLeafErrors walks config and rejects bilingual bags (or other objects)
// on MediaRef string leaves. Used by PUT draft / validate / publish gate.
func CollectMediaRefLeafErrors(config map[string]interface{}) []ValidationError {
	var errors []ValidationError
	walkMediaRefLeaves(config, "", &errors)
	return errors
}

func walkMediaRefLeaves(node interface{}, path string, out *[]ValidationError) {
	switch v := node.(type) {
	case map[string]interface{}:
		if looksLikeMediaRef(v) {
			checkMediaRefLeaves(v, path, out)
			// Still walk nested non-leaf structure if any (unlikely on MediaRef).
		}
		for key, child := range v {
			// Skip re-walking MediaRef string leaves already checked.
			if looksLikeMediaRef(v) {
				if _, isLeaf := mediaLeafKeys[key]; isLeaf {
					continue
				}
			}
			childPath := joinPath(path, key)
			// Named media slots: force MediaRef object shape.
			if _, isSlot := mediaSlotKeys[key]; isSlot {
				if m, ok := child.(map[string]interface{}); ok {
					checkMediaRefLeaves(m, childPath, out)
					// Do not double-walk the same media object via looksLikeMediaRef.
					continue
				}
				if child != nil {
					*out = append(*out, ValidationError{
						Path:    childPath,
						Code:    "MEDIAREF_TYPE",
						Message: "Media reference must be an object with string url/alt/caption",
					})
				}
				continue
			}
			walkMediaRefLeaves(child, childPath, out)
		}
	case []interface{}:
		for i, item := range v {
			itemPath := path
			if path == "" {
				itemPath = fmt.Sprintf("[%d]", i)
			} else {
				itemPath = fmt.Sprintf("%s[%d]", path, i)
			}
			// Showcase items are often bare MediaRef objects in an array.
			if m, ok := item.(map[string]interface{}); ok && looksLikeMediaRef(m) {
				checkMediaRefLeaves(m, itemPath, out)
				continue
			}
			walkMediaRefLeaves(item, itemPath, out)
		}
	}
}

func looksLikeMediaRef(m map[string]interface{}) bool {
	if _, hasURL := m["url"]; hasURL {
		return true
	}
	_, hasAlt := m["alt"]
	_, hasCaption := m["caption"]
	// alt/caption alone without other structural keys — treat as media leaf bag
	if (hasAlt || hasCaption) && len(m) <= 3 {
		// exclude LocalizedText {zh,en}
		if isLocalizedBag(m) {
			return false
		}
		return true
	}
	return false
}

func isLocalizedBag(m map[string]interface{}) bool {
	if len(m) == 0 {
		return false
	}
	for k := range m {
		if k != "zh" && k != "en" {
			return false
		}
	}
	return true
}

func checkMediaRefLeaves(m map[string]interface{}, path string, out *[]ValidationError) {
	for key := range mediaLeafKeys {
		val, ok := m[key]
		if !ok || val == nil {
			continue
		}
		switch val.(type) {
		case string:
			// ok
		default:
			*out = append(*out, ValidationError{
				Path:    joinPath(path, key),
				Code:    "MEDIAREF_TYPE",
				Message: fmt.Sprintf("%s must be a string (not a bilingual object or other type)", key),
			})
		}
	}
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	if strings.HasSuffix(base, "]") {
		return base + "." + key
	}
	return base + "." + key
}
