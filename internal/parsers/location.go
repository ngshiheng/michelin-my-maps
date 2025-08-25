package parsers

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

/*
parseCoordinates extracts latitude and longitude from a Michelin Guide JSON-LD script.
Returns the coordinates as strings to match the database schema.
Returns empty strings if coordinates are not found or invalid.
*/
func parseCoordinates(jsonLD string) (latitude, longitude string) {
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

	if latValue, ok := ld["latitude"]; ok {
		if lat, valid := parseCoordinate(latValue); valid {
			latitude = lat
		}
	}

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

// ParseLocationFromAddress extracts location information from address string
// return the last part as location (usually city/country)
// FIXME: could be enhanced with more sophisticated parsing
func ParseLocationFromAddress(address string) string {
	parts := strings.Split(address, ",")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return ""
}

// ExtractCoordinates attempts to extract latitude and longitude using multiple methods
func ExtractCoordinates(e *colly.XMLElement) (lat, lng string) {
	// Method 1: Try JSON-LD extraction first (highest priority)
	jsonLDSelectors := AwardSelectors["publishedDate"] // Reuse the JSON-LD selectors
	for _, selector := range jsonLDSelectors {
		if jsonLD := e.ChildText(selector); jsonLD != "" {
			if latitude, longitude := parseCoordinates(jsonLD); latitude != "" && longitude != "" {
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
