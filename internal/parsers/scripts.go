package parsers

import (
	"regexp"
)

// ParseDLayerValue parses a value from a dLayer script.
// Supported: Only extracts from assignment syntax, not object literals.
// Example (supported):
//
// script := "dLayer['distinction'] = '3 star';"
// value := ParseDLayerValue(script, "distinction")
// // value == "3 star"
//
// Example (not supported):
//
// script := "dLayer = { 'distinction': '1 star' };"
// value := ParseDLayerValue(script, "distinction")
// // value == ""
//
// To support object literal syntax, the parsing logic must be extended.
func ParseDLayerValue(script, key string) string {
	re := regexp.MustCompile(key + `'\]\s*=\s*'([^']*)'`)
	m := re.FindStringSubmatch(script)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
