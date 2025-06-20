package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitUnpack(t *testing.T) {
	cases := []struct {
		Separator string
		Got       string
		Price     string
		Cuisine   string
	}{
		{"•", "145 - 195 EUR • Modern Cuisine, Classic Cuisin", "145 - 195 EUR", "Modern Cuisine, Classic Cuisin"},
		{"•", "45 - 160 CHF • Italian Contemporary", "45 - 160 CHF", "Italian Contemporary"},
		{"•", "75 - 115 EUR • Piedmontese, Creative", "75 - 115 EUR", "Piedmontese, Creative"},
		{"•", "70 EUR • Regional Cuisine", "70 EUR", "Regional Cuisine"},
		{"•", "31,000 JPY • Innovative", "31,000 JPY", "Innovative"},
		{"•", "25,000 - 28,000 JPY • Sushi", "25,000 - 28,000 JPY", "Sushi"},
		{"•", "European Contemporary", "", "European Contemporary"},
		{"•", "$$$$ • French", "$$$$", "French"},
		{"·", "¥¥¥ · Japanese", "¥¥¥", "Japanese"},
	}

	for _, tt := range cases {
		t.Run("test SplitUnpack", func(t *testing.T) {
			price, cuisine := SplitUnpack(tt.Got, tt.Separator)
			assert.Equal(t, tt.Price, price)
			assert.Equal(t, tt.Cuisine, cuisine)
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

func TestParseDistinction(t *testing.T) {
	cases := []struct {
		Got      string
		Expected string
	}{
		{"Two Stars: Excellent cooking", "2 Stars"},
		{"One Star: High quality cooking", "1 Star"},
		{"Three Stars: Exceptional cuisine", "3 Stars"},
		{"Bib Gourmand: good quality, good value cooking", "Bib Gourmand"},
		{"MICHELIN Green Star", "Selected Restaurants"},
		{"Street Food", "Selected Restaurants"},
		{"Invalid Input", "Selected Restaurants"},
	}

	for _, tt := range cases {
		t.Run("test ParseDistinction", func(t *testing.T) {
			got := ParseDistinction(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}
