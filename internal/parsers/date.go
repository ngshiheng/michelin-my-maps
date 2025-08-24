package parsers

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

func parsePublishedDate(text string) string {
	// Try JSON-LD first
	if year := ParsePublishedYear(text); year != 0 {
		return strconv.Itoa(year)
	}

	// Try text parsing
	if date := ParseDateFromText(text); date != "" {
		return date
	}

	return text
}

func parseYear(dateText string) int {
	if dateText == "" {
		return 0
	}

	// Try to parse as integer first
	if year, err := strconv.Atoi(dateText); err == nil && year > 1900 && year < 3000 {
		return year
	}

	// Try extraction from any format
	return ParseYearFromAnyFormat(dateText)
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

/*
ParseCoordinates extracts latitude and longitude from a Michelin Guide JSON-LD script.
Returns the coordinates as strings to match the database schema.
Returns empty strings if coordinates are not found or invalid.
*/
func ParseCoordinates(jsonLD string) (latitude, longitude string) {
	if jsonLD == "" {
		return "", ""
	}

	var ld map[string]any
	if err := json.Unmarshal([]byte(jsonLD), &ld); err != nil {
		return "", ""
	}

	parseCoordinate := func(value any) (string, bool) {
		switch v := value.(type) {
		case string:
			if v != "" && validateCoordinate(v) {
				return v, true
			}
		case float64:
			coordStr := strconv.FormatFloat(v, 'f', -1, 64)
			if validateCoordinate(coordStr) {
				return coordStr, true
			}
		case int:
			coordStr := strconv.Itoa(v)
			if validateCoordinate(coordStr) {
				return coordStr, true
			}
		}
		return "", false
	}

	// Extract latitude
	if latValue, ok := ld["latitude"]; ok {
		if lat, valid := parseCoordinate(latValue); valid {
			latitude = lat
		}
	}

	// Extract longitude
	if lngValue, ok := ld["longitude"]; ok {
		if lng, valid := parseCoordinate(lngValue); valid {
			longitude = lng
		}
	}

	return latitude, longitude
}

// validateCoordinate checks if a coordinate string represents a valid latitude or longitude.
func validateCoordinate(coordStr string) bool {
	coord, err := strconv.ParseFloat(coordStr, 64)
	if err != nil {
		return false
	}

	// Basic validation: coordinates should be reasonable values
	// Latitude: -90 to 90, Longitude: -180 to 180
	// We'll do a broader check here since we don't know which is which
	return coord >= -180.0 && coord <= 180.0
}
