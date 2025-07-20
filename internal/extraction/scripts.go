package extraction

import (
	"regexp"

	"github.com/nyaruka/phonenumbers"
)

// ParsePhoneNumber extracts and parses phone number from a raw string.
// Example inputPhoneNumber: "+81 3-3874-1552"
func ParsePhoneNumber(inputPhoneNumber string) string {
	parsedPhoneNumber, err := phonenumbers.Parse(inputPhoneNumber, "")
	if err != nil {
		return ""
	}

	return phonenumbers.Format(parsedPhoneNumber, phonenumbers.E164)
}

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
