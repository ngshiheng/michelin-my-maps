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
				t.Errorf("ParseCoordinates(%s) latitude = %q; want %q", tt.name, lat, tt.expectedLat)
			}
			if lng != tt.expectedLng {
				t.Errorf("ParseCoordinates(%s) longitude = %q; want %q", tt.name, lng, tt.expectedLng)
			}
		})
	}
}

func TestValidateCoordinate(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"57.0649323", true},
		{"12.5594919", true},
		{"-122.456", true},
		{"0", true},
		{"180", true},
		{"-180", true},
		{"181", false},
		{"-181", false},
		{"not-a-number", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := validateCoordinate(tt.input)
			if got != tt.expected {
				t.Errorf("validateCoordinate(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}
