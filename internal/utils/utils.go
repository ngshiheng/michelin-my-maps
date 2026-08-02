package utils

import (
	"net/http"
	"strings"
)

// FlattenHeaders converts http.Header into a simple map[string]string for structured logging.
// Multiple values for the same header key are joined with "; "
func FlattenHeaders(h *http.Header) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string, len(*h))
	for k, v := range *h {
		out[k] = strings.Join(v, "; ")
	}
	return out
}

func FieldsToArgs(fields map[string]any) []any {
	args := make([]any, 0, len(fields)*2)
	for key, value := range fields {
		args = append(args, key, value)
	}
	return args
}
