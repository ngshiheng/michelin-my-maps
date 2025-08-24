package parsers

import (
	"strings"
)

// NormalizeAddress cleans up address strings by removing newlines and normalizing whitespace.
// Original data examples: "Shaw Centre, #01-16,\n1 Scotts Road, 228208, Singapore"
func NormalizeAddress(address string) string {
	// Replace newlines with spaces
	normalized := strings.ReplaceAll(address, "\n", " ")

	// Normalize multiple spaces to single space
	normalized = strings.Join(strings.Fields(normalized), " ")

	return strings.TrimSpace(normalized)
}

// TrimWhiteSpaces removes various whitespace characters including line breaks and multiple spaces.
func TrimWhiteSpaces(str string) string {
	if str == "" {
		return ""
	}

	// Remove line breaks and normalize spaces
	trimmed := strings.ReplaceAll(str, "\n", "")
	trimmed = strings.ReplaceAll(trimmed, "  ", " ")

	return strings.TrimSpace(trimmed)
}

// ExtractTextFromElements extracts non-empty text from XPath results, trimming whitespace.
func ExtractTextFromElements(texts []string) string {
	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text != "" {
			return text
		}
	}
	return ""
}

// JoinFacilities joins facility strings with a consistent separator, filtering out empty values.
// Original data examples: ["Air conditioning", "", "Car park", "Interesting wine list"]
func JoinFacilities(facilities []string) string {
	var nonEmpty []string
	for _, facility := range facilities {
		if trimmed := strings.TrimSpace(facility); trimmed != "" {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}
	return strings.Join(nonEmpty, ",")
}

// SplitUnpack performs SplitN and unpacks a string.
func SplitUnpack(str string, separator string) (string, string) {
	if len(str) == 0 {
		return str, str
	}

	parsedStr := strings.SplitN(str, separator, 2)

	for i, s := range parsedStr {
		parsedStr[i] = strings.TrimSpace(s)
	}

	if len(parsedStr) == 1 {
		return "", parsedStr[0] // Always assume price is missing
	}

	return parsedStr[0], parsedStr[1]
}

// SplitUnpackMultiDelimiter attempts to split a string using multiple possible delimiters.
// Tries delimiters in order and returns the first successful split.
// Original data examples:
//
//	"$$$ · French" (middle dot)
//	"$$$ • French" (bullet)
//	"$$$ - French" (hyphen)
//	"$$$ | French" (pipe)
func SplitUnpackMultiDelimiter(str string, delimiters []string) (string, string) {
	if len(str) == 0 {
		return str, str
	}

	// Try each delimiter in order
	for _, delimiter := range delimiters {
		if strings.Contains(str, delimiter) {
			return SplitUnpack(str, delimiter)
		}
	}

	// If no delimiter found, assume it's all cuisine (price missing)
	return "", strings.TrimSpace(str)
}
