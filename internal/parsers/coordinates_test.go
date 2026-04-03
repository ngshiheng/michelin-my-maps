package parsers

import (
	"testing"
)

func TestParseCoordinates(t *testing.T) {
	tests := []struct {
		name        string
		jsonLD      string
		expectedLat string
		expectedLng string
	}{
		{
			name:        "valid coordinates as numbers",
			jsonLD:      `{"latitude":57.0649323,"longitude":12.5594919}`,
			expectedLat: "57.0649323",
			expectedLng: "12.5594919",
		},
		{
			name:        "valid coordinates as strings",
			jsonLD:      `{"latitude":"1.3521","longitude":"103.8198"}`,
			expectedLat: "1.3521",
			expectedLng: "103.8198",
		},
		{
			name:        "full Michelin JSON-LD example",
			jsonLD:      `{"@context":"http://schema.org","name":"NG","latitude":57.0649323,"longitude":12.5594919,"review":{"datePublished":"2025-06-16T12:11"},"award":{"dateAwarded":"2025"}}`,
			expectedLat: "57.0649323",
			expectedLng: "12.5594919",
		},
		{
			name:        "missing coordinates",
			jsonLD:      `{"name":"Restaurant"}`,
			expectedLat: "",
			expectedLng: "",
		},
		{
			name:        "only latitude",
			jsonLD:      `{"latitude":45.123}`,
			expectedLat: "45.123",
			expectedLng: "",
		},
		{
			name:        "only longitude",
			jsonLD:      `{"longitude":-122.456}`,
			expectedLat: "",
			expectedLng: "-122.456",
		},
		{
			name:        "invalid coordinates (out of range)",
			jsonLD:      `{"latitude":200,"longitude":400}`,
			expectedLat: "",
			expectedLng: "",
		},
		{
			name:        "empty JSON-LD",
			jsonLD:      "",
			expectedLat: "",
			expectedLng: "",
		},
		{
			name:        "invalid JSON",
			jsonLD:      `{invalid json}`,
			expectedLat: "",
			expectedLng: "",
		},
		{
			name:        "coordinates as integers",
			jsonLD:      `{"latitude":45,"longitude":-122}`,
			expectedLat: "45",
			expectedLng: "-122",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lng := parseCoordinates(tt.jsonLD)
			if lat != tt.expectedLat {
				t.Errorf("parseCoordinates(%s) latitude = %q; want %q", tt.name, lat, tt.expectedLat)
			}
			if lng != tt.expectedLng {
				t.Errorf("parseCoordinates(%s) longitude = %q; want %q", tt.name, lng, tt.expectedLng)
			}
		})
	}
}

func TestParseGoogleMapsCoordinates(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantLat string
		wantLng string
	}{
		{
			name:    "valid embed URL",
			src:     "https://www.google.com/maps/embed/v1/place?key=API_KEY&q=51.5078582,-0.7017529",
			wantLat: "51.5078582",
			wantLng: "-0.7017529",
		},
		{
			name:    "negative lat/lng",
			src:     "https://www.google.com/maps/embed/v1/place?key=KEY&q=-33.8688,-151.2093",
			wantLat: "-33.8688",
			wantLng: "-151.2093",
		},
		{
			name:    "out-of-range coordinates",
			src:     "https://www.google.com/maps/embed/v1/place?key=K&q=999.0,-999.0",
			wantLat: "",
			wantLng: "",
		},
		{
			name:    "missing q param",
			src:     "https://www.google.com/maps/embed/v1/place?key=API_KEY",
			wantLat: "",
			wantLng: "",
		},
		{
			name:    "empty string",
			src:     "",
			wantLat: "",
			wantLng: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lng := parseGoogleMapsCoordinates(tt.src)
			if lat != tt.wantLat || lng != tt.wantLng {
				t.Errorf("parseGoogleMapsCoordinates(%q) = (%q, %q); want (%q, %q)", tt.src, lat, lng, tt.wantLat, tt.wantLng)
			}
		})
	}
}
