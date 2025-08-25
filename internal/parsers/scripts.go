package parsers

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
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

// FindDLayerScript searches for a dLayer script containing restaurant distinction data.
func FindDLayerScript(e *colly.XMLElement) string {
	return findScript(e, func(text string) bool {
		return strings.Contains(text, "dLayer") && strings.Contains(text, "distinction")
	})
}

// findScript searches for a <script> tag whose content matches the given condition.
func findScript(e *colly.XMLElement, condition func(string) bool) string {
	scripts := e.ChildTexts("//script")
	for _, script := range scripts {
		if condition(script) {
			return script
		}
	}
	return ""
}
