package util

import (
	"encoding/json"
	"fmt"
)

// PrettyPrint returns a pretty-printed JSON string of the given value.
func PrettyPrint(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error in PrettyPrint: %v", err)
	}
	return string(b)
}
