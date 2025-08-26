package parsers

import (
	"strings"
)

// ParseLocationFromAddress extracts location information from address string
// return the last part as location (usually city/country)
// FIXME: could be enhanced with more sophisticated parsing
func ParseLocationFromAddress(address string) string {
	parts := strings.Split(address, ",")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return ""
}
