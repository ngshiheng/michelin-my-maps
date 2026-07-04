package parsers

import (
	"testing"
)

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
