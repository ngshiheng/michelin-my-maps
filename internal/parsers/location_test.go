package parsers

import (
	"testing"
)

func TestParseLocationFromAddress(t *testing.T) {
	tests := []struct {
		address  string
		expected string
	}{
		{
			address:  "Lieu-dit la Baquère, Préneron, 32190, France",
			expected: "Préneron, France",
		},
		{
			address:  "57 Porte des Ardennes, Erpeldange, 9145, Luxembourg",
			expected: "Erpeldange, Luxembourg",
		},
		{
			address:  "Klara Norra Kyrkogata 26, Stockholm",
			expected: "Stockholm",
		},
		{
			address:  "Hong Kong",
			expected: "Hong Kong, Hong Kong SAR China",
		},
		{
			address:  "Singapore",
			expected: "Singapore",
		},
		{
			address:  "Dubai",
			expected: "Dubai",
		},
		{
			address:  "Macau",
			expected: "Macau",
		},
		{
			address:  "Some Street, Some City, 12345, Some Country",
			expected: "Some City, Some Country",
		},
		{
			address:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		got := ParseLocationFromAddress(tt.address)
		if got != tt.expected {
			t.Errorf("ParseLocationFromAddress(%q) = %q; want %q", tt.address, got, tt.expected)
		}
	}
}
