package parsers

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
)

// parseLocationFromAddress extracts location information from address string
// return the last part as location (usually city/country)
// FIXME: could be enhanced with more sophisticated parsing
func parseLocationFromAddress(address string) string {
	parts := strings.Split(address, ",")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return ""
}

// extractCoordinates attempts to extract latitude and longitude using multiple methods
func extractCoordinates(e *colly.XMLElement) (lat, lng string) {
	// Method 1: Try JSON-LD extraction first (highest priority)
	jsonLDSelectors := AwardSelectors["publishedDate"] // Reuse the JSON-LD selectors
	for _, selector := range jsonLDSelectors {
		if jsonLD := e.ChildText(selector); jsonLD != "" {
			if latitude, longitude := ParseCoordinates(jsonLD); latitude != "" && longitude != "" {
				return latitude, longitude
			}
		}
	}

	// Method 2: Try data attributes on elements
	lat = tryRestaurantSelectorsAttr(e, "coordinates", "data-lat")
	lng = tryRestaurantSelectorsAttr(e, "coordinates", "data-lng")
	if lat != "" && lng != "" {
		return lat, lng
	}

	// Method 3: Try Google Maps iframe extraction
	selectors := RestaurantSelectors["googleMaps"]
	for _, selector := range selectors {
		if iframeSrc := e.ChildAttr(selector, "src"); iframeSrc != "" {
			if latitude, longitude := parseGoogleMapsCoordinates(iframeSrc); latitude != "" && longitude != "" {
				return latitude, longitude
			}
		}
	}

	return "", ""
}

// parseGoogleMapsCoordinates extracts lat/lng from Google Maps iframe src
func parseGoogleMapsCoordinates(src string) (lat, lng string) {
	// Google Maps iframe URLs typically contain coordinates in various formats
	// Example: https://maps.google.com/maps?q=1.234567,103.123456
	coordPattern := regexp.MustCompile(`[?&]q=([0-9.-]+),([0-9.-]+)`)
	matches := coordPattern.FindStringSubmatch(src)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	// Alternative pattern: center=lat,lng
	centerPattern := regexp.MustCompile(`center=([0-9.-]+),([0-9.-]+)`)
	matches = centerPattern.FindStringSubmatch(src)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	return "", ""
}
