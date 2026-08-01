package contentslots

import (
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func validationErrorsToPathErrors(err error) []PathError {
	if err == nil {
		return nil
	}
	// Prefer structured *ValidationError when available.
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		return flattenValidationError(ve)
	}
	// Fallback: single message
	return []PathError{{
		Path: "(root)", Code: "SCHEMA", Message: err.Error(),
	}}
}

func flattenValidationError(ve *jsonschema.ValidationError) []PathError {
	if ve == nil {
		return nil
	}
	var out []PathError
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if e == nil {
			return
		}
		if len(e.Causes) == 0 {
			path := instancePathString(e)
			msg := e.Error()
			// trim verbose prefix when possible
			if i := strings.Index(msg, "doesn't validate"); i > 0 {
				// keep full error; library messages are useful
			}
			out = append(out, PathError{
				Path:    path,
				Code:    "SCHEMA",
				Message: msg,
			})
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	if len(out) == 0 {
		out = append(out, PathError{
			Path: "(root)", Code: "SCHEMA", Message: ve.Error(),
		})
	}
	return out
}

func instancePathString(e *jsonschema.ValidationError) string {
	if e == nil {
		return "(root)"
	}
	// InstanceLocation is []string path segments in v6
	loc := e.InstanceLocation
	if len(loc) == 0 {
		return "(root)"
	}
	var b strings.Builder
	for i, seg := range loc {
		if i > 0 {
			// array indices are plain numbers in JSON pointer segments
			if isAllDigits(seg) {
				b.WriteByte('[')
				b.WriteString(seg)
				b.WriteByte(']')
				continue
			}
			b.WriteByte('.')
		}
		if isAllDigits(seg) && i == 0 {
			b.WriteByte('[')
			b.WriteString(seg)
			b.WriteByte(']')
			continue
		}
		b.WriteString(seg)
	}
	s := b.String()
	if s == "" {
		return "(root)"
	}
	return s
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

