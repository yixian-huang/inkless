package contentslots

import "encoding/json"

func jsonMarshalImpl(v any) ([]byte, error) {
	return json.Marshal(v)
}
