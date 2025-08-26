// Package parsers provides utilities for extracting published year data from restaurant HTML.
package parsers

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

var (
	yearMichelinGuideRegex = regexp.MustCompile(`(\d{4})\s+MICHELIN Guide`)
	michelinGuideYearRegex = regexp.MustCompile(`MICHELIN Guide.*?(\d{4})`)
	isoDateRegex           = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	datePatterns           = []*regexp.Regexp{yearMichelinGuideRegex, michelinGuideYearRegex, isoDateRegex}
	commonDateLayouts      = []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	dateXPath = "//div[contains(@class,'restaurant-details__heading--label-title')] | //div[contains(@class,'label-text')]"
	metaXPath = "//meta[@name='description']"
)

// ExtractPublishedYear tries JSON-LD, then XPath, then meta, returning the first valid year.
func ExtractPublishedYear(e *colly.XMLElement) int {
	if year := extractYearFromJSONLD(e); year != 0 {
		return year
	}
	if year := extractYearFromXPath(e, dateXPath); year != 0 {
		return year
	}
	if year := extractYearFromMeta(e, metaXPath, "content"); year != 0 {
		return year
	}
	return 0
}

// extractYearFromJSONLD extracts the published year from a JSON-LD <script> tag containing restaurant data.
//
// Example input (script content):
//
//	{
//	  "@context":"http://schema.org",
//	  "@type":"Restaurant",
//	  "award":{"dateAwarded":"2023-01-25"},
//	  "review":{"datePublished":"2022-02-16T08:02"}
//	}
//
// Example usage:
//
//	year := extractYearFromJSONLD(e) // returns 2023 (from award.dateAwarded), or 2022 if only review.datePublished is present
func extractYearFromJSONLD(e *colly.XMLElement) int {
	jsonLD := findScriptByKeywords(e, `"@type":"Restaurant"`)
	if jsonLD == "" {
		return 0
	}
	return parsePublishedYear(jsonLD)
}

// extractYearFromXPath extracts year from text content at the given XPath.
func extractYearFromXPath(e *colly.XMLElement, xpath string) int {
	texts := e.ChildTexts(xpath)
	for _, text := range texts {
		if year := parseYearFromAnyFormat(strings.TrimSpace(text)); year != 0 {
			return year
		}
	}
	return 0
}

// extractYearFromMeta extracts year from a meta tag's attribute.
func extractYearFromMeta(e *colly.XMLElement, xpath, attr string) int {
	metaContent := e.ChildAttr(xpath, attr)
	if metaContent != "" {
		return parseYearFromAnyFormat(metaContent)
	}
	return 0
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
	if yearStr := parseDateFromText(text); len(yearStr) == 4 {
		if year, err := strconv.Atoi(yearStr); err == nil {
			return year
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

// parseDateFromText attempts to parse a date from text using all known date patterns.
func parseDateFromText(text string) string {
	if text == "" {
		return ""
	}
	text = strings.TrimSpace(text)
	for _, pattern := range datePatterns {
		if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// parsePublishedYear extracts the year from a Michelin Guide JSON-LD script.
// Extraction priority:
//  1. award.dateAwarded (if present and valid)
//  2. review.datePublished (fallback)
func parsePublishedYear(jsonLD string) int {
	var ld map[string]any
	if err := json.Unmarshal([]byte(jsonLD), &ld); err != nil {
		return 0
	}

	parseYear := func(s string) (int, bool) {
		if len(s) == 4 {
			if year, err := strconv.Atoi(s); err == nil && validateYear(year) {
				return year, true
			}
		}
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
	// Try award.dateAwarded
	if award, ok := ld["award"].(map[string]any); ok {
		if dateAwarded, ok := award["dateAwarded"].(string); ok && dateAwarded != "" {
			if year, ok := parseYear(dateAwarded); ok {
				return year
			}
		}
	}
	// Try review.datePublished
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
func validateYear(year int) bool {
	currentYear := time.Now().Year()
	return year >= 1900 && year <= currentYear+1 // Allow one year in future for edge cases
}

// findScriptByKeywords returns the first <script> tag's text containing all keywords.
func findScriptByKeywords(e *colly.XMLElement, keywords ...string) string {
	scripts := e.ChildTexts("//script")
	for _, script := range scripts {
		found := true
		for _, kw := range keywords {
			if !strings.Contains(script, kw) {
				found = false
				break
			}
		}
		if found {
			return script
		}
	}
	return ""
}
