package parsers

import (
	"testing"

	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
)

func TestParseDistinction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Three Stars", models.ThreeStars},
		{"3 star", models.ThreeStars},
		{"two stars", models.TwoStars},
		{"1 star", models.OneStar},
		{"Bib Gourmand", models.BibGourmand},
		{"Selected Restaurant", models.SelectedRestaurants},
		{"Plate", models.SelectedRestaurants},
		{"", models.SelectedRestaurants},
		{"random text", models.SelectedRestaurants},
		{"&bull; 3 star", models.ThreeStars},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDistinction(tt.input)
			if got != tt.expected {
				t.Errorf("ParseDistinction(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"&bull; 3 star", " 3 star"},
		{"• 2 stars", " 2 stars"},
		{"No entity", "No entity"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DecodeHTMLEntities(tt.input)
			if got != tt.expected {
				t.Errorf("decodeHTMLEntities(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
