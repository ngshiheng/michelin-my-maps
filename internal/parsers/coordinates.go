// Package parsers provides utilities for extracting latitude and longitude from restaurant HTML.
package parsers

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

var googleMapsSelectors = RestaurantSelectors["googleMaps"]

// ExtractCoordinates tries JSON-LD, then Google Maps iframe, returning the first valid lat/lng.
func ExtractCoordinates(e *colly.XMLElement) (lat, lng string) {
	if lat, lng := extractCoordinatesFromJSONLD(findAndParseJSONLD(e)); lat != "" && lng != "" {
		return lat, lng
	}
	if lat, lng := extractCoordinatesFromGoogleMaps(e); lat != "" && lng != "" {
		return lat, lng
	}
	if lat, lng := extractCoordinatesFromMapDiv(e); lat != "" && lng != "" {
		return lat, lng
	}
	return "", ""
}

func extractCoordinatesFromMapDiv(e *colly.XMLElement) (latitude, longitude string) {
	lat := tryRestaurantSelectorsAttr(e, "googleMapDiv", "data-center-lat")
	lng := tryRestaurantSelectorsAttr(e, "googleMapDiv", "data-center-lng")
	if lat == "" || lng == "" {
		return "", ""
	}
	if cLat, err := strconv.ParseFloat(lat, 64); err != nil || cLat < -180.0 || cLat > 180.0 {
		return "", ""
	}
	if cLng, err := strconv.ParseFloat(lng, 64); err != nil || cLng < -180.0 || cLng > 180.0 {
		return "", ""
	}
	return lat, lng
}

func extractCoordinatesFromJSONLD(ld *jsonLDRestaurant) (latitude, longitude string) {
	if ld == nil {
		return "", ""
	}
	return ld.coordinates()
}

func extractCoordinatesFromGoogleMaps(e *colly.XMLElement) (latitude, longitude string) {
	for _, selector := range googleMapsSelectors {
		if iframeSrc := e.ChildAttr(selector, "src"); iframeSrc != "" {
			lat, lng := parseGoogleMapsCoordinates(iframeSrc)
			if lat != "" && lng != "" {
				return lat, lng
			}
		}
	}
	return "", ""
}

// parseGoogleMapsCoordinates extracts latitude and longitude from a Google Maps embed URL.
// e.g.:
//
//	url := "https://www.google.com/maps/embed/v1/place?key=API_KEY&q=51.5078582,-0.7017529"
//	lat, lng := parseGoogleMapsCoordinates(url) // lat == "51.5078582", lng == "-0.7017529"
func parseGoogleMapsCoordinates(src string) (lat, lng string) {
	u, err := url.Parse(src)
	if err != nil {
		return "", ""
	}

	q := u.Query().Get("q")
	parts := strings.Split(q, ",")
	if len(parts) != 2 {
		return "", ""
	}

	latRaw := strings.TrimSpace(parts[0])
	lngRaw := strings.TrimSpace(parts[1])
	if cLat, err := strconv.ParseFloat(latRaw, 64); err == nil && cLat >= -180.0 && cLat <= 180.0 {
		lat = latRaw
	}
	if cLng, err := strconv.ParseFloat(lngRaw, 64); err == nil && cLng >= -180.0 && cLng <= 180.0 {
		lng = lngRaw
	}
	return lat, lng
}
