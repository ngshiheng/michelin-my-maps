package extraction

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// ParseDateFromText attempts to parse a date from text using all known date patterns.
func ParseDateFromText(text string) string {
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

// ParseYearFromAnyFormat extracts year from various date string formats.
// This enhances and consolidates parser.ParseYear with additional parsing logic.
func ParseYearFromAnyFormat(publishedDate string) int {
	if publishedDate == "" {
		return 0
	}

	// Try parsing as full date first
	for _, layout := range commonDateLayouts {
		if t, err := time.Parse(layout, publishedDate); err == nil {
			return t.Year()
		}
	}

	// Try extracting year from text patterns
	if yearStr := ParseDateFromText(publishedDate); yearStr != "" {
		if len(yearStr) == 4 {
			if year, err := strconv.Atoi(yearStr); err == nil {
				return year
			}
		}
	}

	// Fallback: try parsing as 4-digit year string
	if len(publishedDate) == 4 {
		if year, err := strconv.Atoi(publishedDate); err == nil {
			return year
		}
	}

	return 0
}

// ParsePublishedYear extracts the year from a Michelin Guide JSON-LD script.
// This consolidates the logic from parser.ParsePublishedYear.
func ParsePublishedYear(jsonLD string) (int, error) {
	if jsonLD == "" {
		return 0, nil
	}

	var ld map[string]any
	if err := json.Unmarshal([]byte(jsonLD), &ld); err != nil {
		return 0, err
	}

	review, ok := ld["review"].(map[string]any)
	if !ok {
		return 0, nil
	}

	pd, ok := review["datePublished"].(string)
	if !ok || pd == "" {
		return 0, nil
	}

	// Use the shared date layouts
	for _, layout := range commonDateLayouts {
		if t, err := time.Parse(layout, pd); err == nil {
			year := t.Year()
			if validateYear(year) {
				return year, nil
			}
		}
	}

	// Fallback: try parsing as 4-digit year string
	if len(pd) == 4 {
		if year, err := strconv.Atoi(pd); err == nil && validateYear(year) {
			return year, nil
		}
	}

	return 0, nil
}

// validateYear checks if a year value is reasonable for Michelin Guide data.
// Michelin Guide started in 1900, so any year before that is invalid.
func validateYear(year int) bool {
	currentYear := time.Now().Year()
	return year >= 1900 && year <= currentYear+1 // Allow one year in future for edge cases
}
