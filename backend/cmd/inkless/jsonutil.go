package main

import (
	"encoding/json"
	"io"
)

func jsonNewEncoder(w io.Writer) func(any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode
}

func jsonMarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
