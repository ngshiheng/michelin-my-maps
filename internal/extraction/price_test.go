package extraction

import (
	"testing"
)

func TestValidatePriceText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"$$$$", "$$$$"},
		{"€€€", "€€€"},
		{"1,800 NOK", "1,800 NOK"},
		{"155 EUR", "155 EUR"},
		{"300 - 2,000 MOP", "300 - 2,000 MOP"},
		{"155 - 380", "155 - 380"},
		{"Over 75 USD", "Over 75 USD"},
		{"Between 350 and 500 HKD", "Between 350 and 500 HKD"},
		{"500 to 1500 TWD", "500 to 1500 TWD"},
		{"Less than 200 THB", "Less than 200 THB"},
		{"€€€ • Modern European", "€€€"},
		{"", ""},
		{"invalid", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ValidatePriceText(tt.input)
			if got != tt.expected {
				t.Errorf("ValidatePriceText(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMapPrice(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CAT_P01", "$"},
		{"CAT_P02", "$$"},
		{"CAT_P03", "$$$"},
		{"CAT_P04", "$$$$"},
		{"", ""},
		{"random", "random"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapPrice(tt.input)
			if got != tt.expected {
				t.Errorf("MapPrice(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCleanPriceValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$$$$\"`, "$$$$\""},
		{`€€€`, "€€€"},
		{`155\\\"EUR`, "155\"EUR"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CleanPriceValue(tt.input)
			if got != tt.expected {
				t.Errorf("CleanPriceValue(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizePriceText(t *testing.T) {
	tests := []struct {
		input      string
		separators string
		expected   string
	}{
		{"€€€ • Modern European", "·•", "€€€"},
		{"$$$ · French cuisine", "·•", "$$$"},
		{"155 - 380", "·•", "155 - 380"},
		{"  1,800 NOK  ", "·•", "1,800 NOK"},
		{"", "·•", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePriceText(tt.input, tt.separators)
			if got != tt.expected {
				t.Errorf("normalizePriceText(%q, %q) = %q; want %q", tt.input, tt.separators, got, tt.expected)
			}
		})
	}
}
