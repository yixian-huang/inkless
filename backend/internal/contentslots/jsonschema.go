package contentslots

import (
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidateJSONSchema validates config against an inline JSON Schema document.
// Returns empty if schema is nil/empty. Compile/runtime errors become SCHEMA_* PathErrors.
func ValidateJSONSchema(config map[string]any, schema map[string]any) []PathError {
	if schema == nil || len(schema) == 0 {
		return nil
	}
	if config == nil {
		config = map[string]any{}
	}

	// Compiler expects a decoded JSON document; re-round-trip to normalize numbers/maps.
	doc, err := normalizeJSONDoc(schema)
	if err != nil {
		return []PathError{{
			Path: "(schema)", Code: "SCHEMA_INVALID",
			Message: "theme schema is not valid JSON document: " + err.Error(),
		}}
	}
	inst, err := normalizeJSONDoc(config)
	if err != nil {
		return []PathError{{
			Path: "(config)", Code: "SCHEMA_INVALID",
			Message: "config is not a valid JSON document: " + err.Error(),
		}}
	}

	c := jsonschema.NewCompiler()
	const loc = "mem://theme-content-slot.json"
	if err := c.AddResource(loc, doc); err != nil {
		return []PathError{{
			Path: "(schema)", Code: "SCHEMA_COMPILE",
			Message: err.Error(),
		}}
	}
	sch, err := c.Compile(loc)
	if err != nil {
		return []PathError{{
			Path: "(schema)", Code: "SCHEMA_COMPILE",
			Message: err.Error(),
		}}
	}
	if err := sch.Validate(inst); err != nil {
		return validationErrorsToPathErrors(err)
	}
	return nil
}

func normalizeJSONDoc(v any) (any, error) {
	// UnmarshalJSON path used by the library examples for map sources.
	b, err := jsonMarshal(v)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(strings.NewReader(string(b)))
}

// separated for testability / avoid importing encoding/json twice in signature noise
func jsonMarshal(v any) ([]byte, error) {
	return jsonMarshalImpl(v)
}
