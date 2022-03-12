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
	}

	for _, tt := range cases {
		t.Run("test TrimWhiteSpaces", func(t *testing.T) {
			got := TrimWhiteSpaces(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}
