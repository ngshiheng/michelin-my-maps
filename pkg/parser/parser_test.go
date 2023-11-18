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
		{"MICHELIN Green Star", "Green Star"},
		{"Unknown Distinction", ""},
		{"Invalid Input", ""},
	}

	for _, tt := range cases {
		t.Run("test ParseDistinction", func(t *testing.T) {
			got := ParseDistinction(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}

func TestParseCoordinates(t *testing.T) {
	cases := []struct {
		Got       string
		Longitude string
		Latitude  string
	}{
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=41.3906717,2.1695083&language=en-US", "41.3906717", "2.1695083"},
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=35.0067137,135.7760153&language=en-US", "35.0067137", "135.7760153"},
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=-180.5426229,190.0029797&language=en-US", "", ""},
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=Xingnan Avenue, Guangzhou&language=en-US", "", ""},
		{"https://www.google.com/maps/embed/v1/place?key=AIzaSyDvEyVCVpGtn81z5NrMKgdehPsrO9sJiMw&q=&language=en-US", "", ""},
	}

	for _, tt := range cases {
		t.Run("test ParseCoordinates", func(t *testing.T) {
			longitude, latitude := ParseCoordinates(tt.Got)
			assert.Equal(t, tt.Longitude, longitude)
			assert.Equal(t, tt.Latitude, latitude)
		})
	}
}

func TestParsePrice(t *testing.T) {
	cases := []struct {
		Got      string
		MinPrice string
		MaxPrice string
		Currency string
	}{
		{"", "", "", ""},
		{"350-450USD", "350", "450", "USD"},
		{"19,000-53,000JPY", "19,000", "53,000", "JPY"},
		{"65-155EUR", "65", "155", "EUR"},
		{"275CHF", "275", "275", "CHF"},
	}

	for _, tt := range cases {
		t.Run("test ParsePrice", func(t *testing.T) {
			minPrice, maxPrice, currency := ParsePrice(tt.Got)
			assert.Equal(t, tt.MinPrice, minPrice)
			assert.Equal(t, tt.MaxPrice, maxPrice)
			assert.Equal(t, tt.Currency, currency)
		})
	}
}

func TestParsePhoneNumber(t *testing.T) {
	cases := []struct {
		Got      string
		Expected string
	}{
		{"", ""},
		{"some test string", ""},
		{"https://www.queensenglishdc.com/", ""},
		{"+32 2 771 14 47", "+3227711447"},
		{"+81 50-3138-5225", "+815031385225"},
	}

	for _, tt := range cases {
		t.Run("test ParsePhoneNumber", func(t *testing.T) {
			got := ParsePhoneNumber(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}
