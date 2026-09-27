package main

import "encoding/json"

// jsonish renders a default as HCL would read it back: strings quoted, lists and maps as JSON.
func jsonish(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "?"
	}
	return string(b)
}
