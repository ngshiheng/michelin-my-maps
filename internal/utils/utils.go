package utils

import (
	"net/http"
	"strings"
)

// FlattenHeaders converts http.Header into a simple map[string]string for structured logging.
// Multiple values for the same header key are joined with "; ".
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

// FlattenCookies parses the Cookie request header and returns a map of name→value pairs.
func FlattenCookies(h *http.Header) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string)
	for _, line := range h.Values("Cookie") {
		for _, pair := range strings.Split(line, ";") {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return out
}
