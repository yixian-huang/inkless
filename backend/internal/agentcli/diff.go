package agentcli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// DiffOp is a leaf-level change in a deep config diff.
type DiffOp struct {
	Path string `json:"path"`
	Op   string `json:"op"` // added | removed | changed
	From any    `json:"from,omitempty"`
	To   any    `json:"to,omitempty"`
}

// DeepConfigDiff recursively compares left (current) vs right (proposed).
// Returns a map with summary + paths (capped for agent token budgets).
func DeepConfigDiff(left, right map[string]any) map[string]any {
	if left == nil {
		left = map[string]any{}
	}
	if right == nil {
		right = map[string]any{}
	}
	var ops []DiffOp
	walkDiff("", left, right, &ops)

	// Cap path list; keep summary accurate.
	const maxPaths = 200
	truncated := false
	shown := ops
	if len(ops) > maxPaths {
		shown = ops[:maxPaths]
		truncated = true
	}

	added, removed, changed := 0, 0, 0
	for _, op := range ops {
		switch op.Op {
		case "added":
			added++
		case "removed":
			removed++
		case "changed":
			changed++
		}
	}

	return map[string]any{
		"summary": map[string]any{
			"added":   added,
			"removed": removed,
			"changed": changed,
			"total":   len(ops),
		},
		"paths":     shown,
		"truncated": truncated,
		// Back-compat top-level key lists (shallow) for older consumers.
		"shallow": ShallowConfigDiff(left, right),
	}
}

func walkDiff(path string, left, right any, out *[]DiffOp) {
	// Normalize JSON numbers etc. via map[string]any tree
	if equalJSON(left, right) {
		return
	}

	lm, lMap := asMap(left)
	rm, rMap := asMap(right)
	if lMap && rMap {
		keys := map[string]struct{}{}
		for k := range lm {
			keys[k] = struct{}{}
		}
		for k := range rm {
			keys[k] = struct{}{}
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			childPath := joinDiffPath(path, k)
			lv, lOK := lm[k]
			rv, rOK := rm[k]
			switch {
			case !lOK:
				*out = append(*out, DiffOp{Path: childPath, Op: "added", To: truncateValue(rv)})
			case !rOK:
				*out = append(*out, DiffOp{Path: childPath, Op: "removed", From: truncateValue(lv)})
			default:
				walkDiff(childPath, lv, rv, out)
			}
		}
		return
	}

	la, lArr := asSlice(left)
	ra, rArr := asSlice(right)
	if lArr && rArr {
		max := len(la)
		if len(ra) > max {
			max = len(ra)
		}
		// If lengths differ a lot, still emit per-index diffs for the prefix,
		// then added/removed tail.
		for i := 0; i < max; i++ {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if path == "" {
				childPath = fmt.Sprintf("[%d]", i)
			}
			switch {
			case i >= len(la):
				*out = append(*out, DiffOp{Path: childPath, Op: "added", To: truncateValue(ra[i])})
			case i >= len(ra):
				*out = append(*out, DiffOp{Path: childPath, Op: "removed", From: truncateValue(la[i])})
			default:
				walkDiff(childPath, la[i], ra[i], out)
			}
		}
		return
	}

	// Leaf mismatch (type change or primitive change)
	*out = append(*out, DiffOp{
		Path: pathOrRoot(path),
		Op:   "changed",
		From: truncateValue(left),
		To:   truncateValue(right),
	})
}

func pathOrRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func joinDiffPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func asMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	// JSON decode sometimes yields map[string]interface{} already; handle via reflect
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		out := make(map[string]any, rv.Len())
		for _, k := range rv.MapKeys() {
			out[k.String()] = rv.MapIndex(k).Interface()
		}
		return out, true
	}
	return nil, false
}

func asSlice(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}
	if s, ok := v.([]any); ok {
		return s, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

func equalJSON(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return reflect.DeepEqual(a, b)
	}
	return string(ab) == string(bb)
}

func truncateValue(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if len(t) > 120 {
			return t[:117] + "…"
		}
		return t
	case map[string]any, []any:
		raw, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		s := string(raw)
		if len(s) > 200 {
			return s[:197] + "…"
		}
		// Prefer compact JSON string for large structures in path list
		if len(s) > 80 {
			return s
		}
		return t
	default:
		s := fmt.Sprint(t)
		if len(s) > 120 {
			return s[:117] + "…"
		}
		return t
	}
}

// LocalMediaRefIssues scans a config for bilingual MediaRef leaves (client-side preflight).
// Mirrors host CollectMediaRefLeafErrors rules at a lightweight level for dry-run without network.
func LocalMediaRefIssues(config map[string]any) []map[string]string {
	var out []map[string]string
	walkLocalMedia(config, "", &out)
	return out
}

func walkLocalMedia(node any, path string, out *[]map[string]string) {
	switch v := node.(type) {
	case map[string]any:
		if looksLikeMediaRefLocal(v) {
			for _, leaf := range []string{"url", "alt", "caption"} {
				if val, ok := v[leaf]; ok && val != nil {
					if _, isStr := val.(string); !isStr {
						*out = append(*out, map[string]string{
							"path":    joinDiffPath(path, leaf),
							"code":    "MEDIAREF_TYPE",
							"message": leaf + " must be a string (not a bilingual object)",
						})
					}
				}
			}
		}
		for k, child := range v {
			childPath := joinDiffPath(path, k)
			if looksLikeMediaRefLocal(v) {
				if k == "url" || k == "alt" || k == "caption" {
					continue
				}
			}
			// named slots
			if k == "media" || k == "backgroundImage" || k == "image" {
				if m, ok := child.(map[string]any); ok {
					walkLocalMedia(m, childPath, out)
					continue
				}
			}
			walkLocalMedia(child, childPath, out)
		}
	case []any:
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if path == "" {
				itemPath = fmt.Sprintf("[%d]", i)
			}
			walkLocalMedia(item, itemPath, out)
		}
	}
}

func looksLikeMediaRefLocal(m map[string]any) bool {
	if _, ok := m["url"]; ok {
		return true
	}
	_, hasAlt := m["alt"]
	_, hasCap := m["caption"]
	if !(hasAlt || hasCap) || len(m) > 3 {
		return false
	}
	// exclude pure {zh,en}
	for k := range m {
		if k != "zh" && k != "en" && k != "alt" && k != "caption" && k != "url" {
			return false
		}
	}
	// {zh,en} only
	onlyLocale := true
	for k := range m {
		if k != "zh" && k != "en" {
			onlyLocale = false
			break
		}
	}
	if onlyLocale {
		return false
	}
	return true
}

// FormatDiffHuman prints a short human summary for non-JSON CLI output.
func FormatDiffHuman(diff map[string]any) string {
	if diff == nil {
		return ""
	}
	sum, _ := diff["summary"].(map[string]any)
	var b strings.Builder
	fmt.Fprintf(&b, "diff: +%v ~%v -%v (total %v)",
		sum["added"], sum["changed"], sum["removed"], sum["total"])
	if trunc, _ := diff["truncated"].(bool); trunc {
		b.WriteString(" [paths truncated]")
	}
	return b.String()
}
