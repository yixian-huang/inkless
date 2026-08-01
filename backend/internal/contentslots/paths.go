package contentslots

import (
	"fmt"
	"strconv"
	"strings"
)

// PathError is a path-rule violation for validate/publish.
type PathError struct {
	Path    string
	Code    string
	Message string
}

// ValidateConfigAgainstSlot enforces mediaRefPaths / stringPaths / localizedPaths
// and optional JSON Schema (SchemaInline) on config.
func ValidateConfigAgainstSlot(config map[string]any, slot Slot) []PathError {
	if config == nil {
		config = map[string]any{}
	}
	var errs []PathError

	for _, p := range slot.MediaRefPaths {
		for _, node := range resolvePath(config, p) {
			if node == nil {
				continue
			}
			m, ok := node.(map[string]any)
			if !ok {
				// allow map[string]interface{} via conversion
				if m2, ok2 := asStringAnyMap(node); ok2 {
					m = m2
					ok = true
				}
			}
			if !ok {
				errs = append(errs, PathError{
					Path: p, Code: "MEDIAREF_TYPE",
					Message: "expected MediaRef object {url?, alt?, caption?}",
				})
				continue
			}
			for _, leaf := range []string{"url", "alt", "caption"} {
				if v, has := m[leaf]; has && v != nil {
					if _, isStr := v.(string); !isStr {
						errs = append(errs, PathError{
							Path: joinLeaf(p, leaf), Code: "MEDIAREF_TYPE",
							Message: leaf + " must be a string",
						})
					}
				}
			}
		}
	}

	for _, p := range slot.StringPaths {
		for _, node := range resolvePath(config, p) {
			if node == nil {
				continue
			}
			if _, isStr := node.(string); !isStr {
				errs = append(errs, PathError{
					Path: p, Code: "TYPE",
					Message: "expected plain string (not bilingual object)",
				})
			}
		}
	}

	for _, p := range slot.LocalizedPaths {
		for _, node := range resolvePath(config, p) {
			if node == nil {
				continue
			}
			switch node.(type) {
			case string:
				// allowed: already resolved or mono
			case map[string]any:
				m := node.(map[string]any)
				if !isLocaleBag(m) {
					errs = append(errs, PathError{
						Path: p, Code: "LOCALIZED_TYPE",
						Message: "expected LocalizedText {zh?, en?} or string",
					})
				}
			default:
				if m, ok := asStringAnyMap(node); ok {
					if !isLocaleBag(m) {
						errs = append(errs, PathError{
							Path: p, Code: "LOCALIZED_TYPE",
							Message: "expected LocalizedText {zh?, en?} or string",
						})
					}
					continue
				}
				errs = append(errs, PathError{
					Path: p, Code: "LOCALIZED_TYPE",
					Message: "expected LocalizedText {zh?, en?} or string",
				})
			}
		}
	}

	// JSON Schema (when theme provides SchemaInline)
	errs = append(errs, ValidateJSONSchema(config, slot.SchemaInline)...)

	return errs
}

func isLocaleBag(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}
	for k, v := range m {
		if k != "zh" && k != "en" {
			return false
		}
		if v == nil {
			continue
		}
		if _, ok := v.(string); !ok {
			return false
		}
	}
	return true
}

func joinLeaf(path, leaf string) string {
	if path == "" {
		return leaf
	}
	if strings.HasSuffix(path, "[]") {
		// rare: treat as parent
		return strings.TrimSuffix(path, "[]") + "[]." + leaf
	}
	return path + "." + leaf
}

func asStringAnyMap(v any) (map[string]any, bool) {
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m, true
	}
	return nil, false
}

// resolvePath walks a simplified path: "a.b[].c" / "a.b[0].c" / "a.b".
// Missing nodes yield no results (not errors).
func resolvePath(root any, path string) []any {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	segments := splitPath(path)
	cur := []any{root}
	for _, seg := range segments {
		next := []any{}
		for _, node := range cur {
			if node == nil {
				continue
			}
			if seg.isArrayAll {
				// parent is array: expand all elements, then optional key
				arr, ok := asSlice(node)
				if !ok {
					continue
				}
				for _, el := range arr {
					if seg.key == "" {
						next = append(next, el)
					} else if m, ok := asStringAnyMap(el); ok {
						if v, has := m[seg.key]; has {
							next = append(next, v)
						}
					}
				}
				continue
			}
			if seg.index >= 0 {
				arr, ok := asSlice(node)
				if !ok || seg.index >= len(arr) {
					continue
				}
				el := arr[seg.index]
				if seg.key == "" {
					next = append(next, el)
				} else if m, ok := asStringAnyMap(el); ok {
					if v, has := m[seg.key]; has {
						next = append(next, v)
					}
				}
				continue
			}
			// plain key on object
			m, ok := asStringAnyMap(node)
			if !ok {
				continue
			}
			if v, has := m[seg.key]; has {
				next = append(next, v)
			}
		}
		cur = next
		if len(cur) == 0 {
			return nil
		}
	}
	return cur
}

type pathSeg struct {
	key        string
	index      int  // -1 = not index
	isArrayAll bool // true for []
}

func splitPath(path string) []pathSeg {
	// Tokenize by '.' but keep [n] / [] attached to previous key.
	// Examples: hero.media → [{hero},{-1}, {media,-1}]
	// showcase.items[] → [{showcase}, {items, all}]
	// features.items[].media → [{features}, {items, all}, {media}]
	// features.items[].title → after [] we need key on each element — use key on array-all seg empty then next key

	parts := strings.Split(path, ".")
	var segs []pathSeg
	for _, p := range parts {
		if p == "" {
			continue
		}
		// items[] or items[0] or media
		if i := strings.IndexByte(p, '['); i >= 0 {
			key := p[:i]
			rest := p[i:]
			if rest == "[]" {
				// array of objects; key is the array field name
				segs = append(segs, pathSeg{key: key, index: -1})
				segs = append(segs, pathSeg{key: "", index: -1, isArrayAll: true})
				continue
			}
			// [n]
			if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
				idxStr := rest[1 : len(rest)-1]
				idx, err := strconv.Atoi(idxStr)
				if err != nil {
					continue
				}
				segs = append(segs, pathSeg{key: key, index: -1})
				segs = append(segs, pathSeg{key: "", index: idx})
				continue
			}
		}
		segs = append(segs, pathSeg{key: p, index: -1})
	}
	return segs
}

// Fix resolvePath for pattern: map key then array expand.
// For pathSeg {key: "items", index: -1} then {isArrayAll: true}:
// first step takes m["items"], second expands array.
// Current split produces: items → arrayAll. Good.

func asSlice(v any) ([]any, bool) {
	if t, ok := v.([]any); ok {
		return t, true
	}
	return nil, false
}

// FormatPathError converts to a stable string for tests.
func FormatPathError(e PathError) string {
	return fmt.Sprintf("%s:%s:%s", e.Path, e.Code, e.Message)
}
