// Package parsers provides utilities for extracting published year data from restaurant HTML.
package parsers

import (
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
)

const (
	xPathDate = "//div[contains(@class,'restaurant-details__heading--label-title')] | //div[contains(@class,'label-text')]"
	xPathMeta = "//meta[@name='description']"
)

// ExtractPublishedYear tries JSON-LD, then XPath, then meta, returning the first valid year.
func ExtractPublishedYear(e *colly.XMLElement) int {
	if year := extractYearFromJSONLDAward(e); year != 0 {
		return year
	}
	if year := extractYearFromJSONLDReview(e); year != 0 {
		return year
	}
	if year := extractYearFromXPath(e, xPathDate); year != 0 {
		return year
	}
	if year := extractYearFromMeta(e, xPathMeta, "content"); year != 0 {
		return year
	}
	return 0
}

// extractYearFromJSONLDAward extracts the guide year from JSON-LD award metadata when present.
//
// e.g. script content:
//
//	{
//	  "@context":"http://schema.org",
//	  "@type":"Restaurant",
//	  "award":{"dateAwarded":"2023-01-25"},
//	  "review":{"datePublished":"2022-02-16T08:02"}
//	}
func extractYearFromJSONLDAward(e *colly.XMLElement) int {
	if ld := findAndParseJSONLD(e); ld != nil {
		return ld.publishedYear()
	}
	return 0
}

func extractYearFromJSONLDReview(e *colly.XMLElement) int {
	if ld := findAndParseJSONLD(e); ld != nil {
		return ld.reviewPublishedYear()
	}
	return 0
}

func extractYearFromXPath(e *colly.XMLElement, xpath string) int {
	texts := e.ChildTexts(xpath)
	for _, text := range texts {
		if year := parseYearFromAnyFormat(strings.TrimSpace(text)); year != 0 {
			return year
		}
	}
	return 0
}

func extractYearFromMeta(e *colly.XMLElement, xpath, attr string) int {
	metaContent := e.ChildAttr(xpath, attr)
	if metaContent != "" {
		return parseYearFromAnyFormat(metaContent)
	}
	return 0
}

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

func validateYear(year int) bool {
	currentYear := time.Now().Year()
	return year >= 1900 && year <= currentYear+1 // Allow one year in future for edge cases
}
