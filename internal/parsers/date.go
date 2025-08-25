package parsers

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// Date extraction patterns for parsing published dates from various HTML formats
var (
	// yearMichelinGuideRegex matches patterns like "2023 MICHELIN Guide"
	yearMichelinGuideRegex = regexp.MustCompile(`(\d{4})\s+MICHELIN Guide`)

	// michelinGuideYearRegex matches patterns like "MICHELIN Guide ... 2023"
	michelinGuideYearRegex = regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`)

	// isoDateRegex matches ISO date format (e.g., "2023-01-25")
	isoDateRegex = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
)

// Standard time layouts used for parsing date strings
var commonDateLayouts = []string{
	"2006-01-02T15:04:05", // Full ISO with seconds
	"2006-01-02T15:04",    // ISO with hours and minutes
	"2006-01-02",          // Date only
}

/*
ExtractPublishedYear extracts the published year of the Michelin award from a restaurant page.
Extraction order: JSON-LD → XPath text → meta description. Returns 0 if not found.
*/
func ExtractPublishedYear(e *colly.XMLElement) int {
	// 1. Try extracting from JSON-LD
	json := findJSONLDScript(e)
	if year := ParsePublishedYear(json); year != 0 {
		return year
	}

	// 2. Try extracting from XPath text content
	if date := extractDateFromTexts(e); date != "" {
		if year := parseYearFromAnyFormat(date); year != 0 {
			return year
		}
	}

	// 3. Try extracting from meta description
	if meta := extractDateFromMetaContent(e); meta != "" {
		if year := parseYearFromAnyFormat(meta); year != 0 {
			return year
		}
	}

	return 0
}

// extractDateFromTexts extracts date from XPath text content using predefined patterns.
func extractDateFromTexts(e *colly.XMLElement) string {
	texts := e.ChildTexts("//div[contains(@class,'restaurant-details__heading--label-title')] | //div[contains(@class,'label-text')]")
	for _, text := range texts {
		if date := parseDateFromText(strings.TrimSpace(text)); date != "" {
			return date
		}
	}
	return ""
}

// extractDateFromMetaContent extracts date from meta description content.
func extractDateFromMetaContent(e *colly.XMLElement) string {
	metaContent := e.ChildAttr("//meta[@name='description']", "content")
	if metaContent != "" {
		return parseDateFromText(metaContent)
	}
	return ""
}

// findJSONLDScript searches for a JSON-LD script containing restaurant data.
func findJSONLDScript(e *colly.XMLElement) string {
	return findScript(e, func(text string) bool {
		return strings.Contains(text, `"@type":"Restaurant"`)
	})
}

// parseDateFromText attempts to parse a date from text using all known date patterns.
func parseDateFromText(text string) string {
	if text == "" {
		return ""
	}

	text = strings.TrimSpace(text)

	// Try each date pattern in order of specificity
	datePatterns := []*regexp.Regexp{
		yearMichelinGuideRegex,
		michelinGuideYearRegex,
		isoDateRegex,
	}

	for _, pattern := range datePatterns {
		if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// parseYearFromAnyFormat extracts year from various date string formats.
func parseYearFromAnyFormat(text string) int {
	if text == "" {
		return 0
	}

	// Try parsing as full date first
	for _, layout := range commonDateLayouts {
		if t, err := time.Parse(layout, text); err == nil {
			return t.Year()
		}
	}

	// Try extracting year from text patterns
	if yearStr := parseDateFromText(text); yearStr != "" {
		if len(yearStr) == 4 {
			if year, err := strconv.Atoi(yearStr); err == nil {
				return year
			}
		}
	}

	// Fallback: try parsing as 4-digit year string
	if len(text) == 4 {
		if year, err := strconv.Atoi(text); err == nil {
			return year
		}
	}

	return 0
}

/*
ParsePublishedYear extracts the year from a Michelin Guide JSON-LD script.
It prioritizes extracting from award.dateAwarded if present and valid,
otherwise falls back to review.datePublished.
*/
func ParsePublishedYear(jsonLD string) int {
	if jsonLD == "" {
		return 0
	}

	var ld map[string]any
	if err := json.Unmarshal([]byte(jsonLD), &ld); err != nil {
		return 0
	}

	parseYear := func(s string) (int, bool) {
		// Try as 4-digit year
		fourDigitYear := len(s) == 4 && strings.TrimSpace(s) != ""
		if fourDigitYear {
			if year, err := strconv.Atoi(s); err == nil && validateYear(year) {
				return year, true
			}
		}
		// Try as date with known layouts
		for _, layout := range commonDateLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				year := t.Year()
				if validateYear(year) {
					return year, true
				}
			}
		}
		return 0, false
	}

	// 1. Try award.dateAwarded first
	if award, ok := ld["award"].(map[string]any); ok {
		if dateAwarded, ok := award["dateAwarded"].(string); ok && dateAwarded != "" {
			if year, ok := parseYear(dateAwarded); ok {
				return year
			}
		}
	}

	// 2. Fallback to review.datePublished
	if review, ok := ld["review"].(map[string]any); ok {
		if pd, ok := review["datePublished"].(string); ok && pd != "" {
			if year, ok := parseYear(pd); ok {
				return year
			}
		}
	}

	return 0
}

// validateYear checks if a year value is reasonable for Michelin Guide data.
// Michelin Guide started in 1900, so any year before that is invalid.
func validateYear(year int) bool {
	currentYear := time.Now().Year()
	return year >= 1900 && year <= currentYear+1 // Allow one year in future for edge cases
}
