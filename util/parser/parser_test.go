package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitUnpack(t *testing.T) {
	cases := []struct {
		Got   string
		Price string
		Type  string
	}{
		{"145 - 195 EUR • Modern Cuisine, Classic Cuisin", "145 - 195 EUR", "Modern Cuisine, Classic Cuisin"},
		{"45 - 160 CHF • Italian Contemporary", "45 - 160 CHF", "Italian Contemporary"},
		{"75 - 115 EUR • Piedmontese, Creative", "75 - 115 EUR", "Piedmontese, Creative"},
		{"70 EUR • Regional Cuisine", "70 EUR", "Regional Cuisine"},
		{"31,000 JPY • Innovative", "31,000 JPY", "Innovative"},
		{"25,000 - 28,000 JPY • Sushi", "25,000 - 28,000 JPY", "Sushi"},
		{"European Contemporary", "", "European Contemporary"},
	}

	for _, tt := range cases {
		t.Run("test split SplitUnack", func(t *testing.T) {
			got1, got2 := SplitUnpack(tt.Got, "•")
			assert.Equal(t, tt.Price, got1)
			assert.Equal(t, tt.Type, got2)
		})
	}
}

func TestTrimWhiteSpaces(t *testing.T) {
	cases := []struct {
		Got      string
		Expected string
	}{
		{"\n                                    Bib Gourmand: good quality, good value cooking\n                                ", "Bib Gourmand: good quality, good value cooking"},
		{"\n                                    One MICHELIN Star: High quality cooking, worth a stop!\n                                ", "One MICHELIN Star: High quality cooking, worth a stop!"},
		{"\n                                    Two MICHELIN Stars: Excellent cooking, worth a detour!\n                                ", "Two MICHELIN Stars: Excellent cooking, worth a detour!"},
		{"\n                                    Three MICHELIN Stars: Exceptional cuisine, worth a special journey!\n                                ", "Three MICHELIN Stars: Exceptional cuisine, worth a special journey!"},
	}

	for _, tt := range cases {
		t.Run("test TrimWhiteSpaces", func(t *testing.T) {
			got := TrimWhiteSpaces(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}

func TestExtractCoordinates(t *testing.T) {
	cases := []struct {
		Got       string
		Longitude string
		Latitude  string
	}{
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=41.3906717,2.1695083&language=en-US", "41.3906717", "2.1695083"},
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=35.0067137,135.7760153&language=en-US", "35.0067137", "135.7760153"},
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=53.5426229,10.0029797&language=en-US", "53.5426229", "10.0029797"},
	}

	for _, tt := range cases {
		t.Run("test TrimWhiteSpaces", func(t *testing.T) {
			longitude, latitude := ExtractCoordinates(tt.Got)
			assert.Equal(t, tt.Longitude, longitude)
			assert.Equal(t, tt.Latitude, latitude)
		})
	}
}
