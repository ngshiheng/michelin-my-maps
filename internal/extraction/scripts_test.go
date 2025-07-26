package extraction

import (
	"testing"
)

func TestParsePhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+81 3-3874-1552", "+81338741552"},
		{"+1 415-555-2671", "+14155552671"},
		{"not a phone", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParsePhoneNumber(tt.input)
			if got != tt.expected {
				t.Errorf("ParsePhoneNumber(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseDLayerValue(t *testing.T) {
	tests := []struct {
		script   string
		key      string
		expected string
	}{
		{"dLayer['distinction'] = '3 star';", "distinction", "3 star"},
		{"dLayer['price'] = '155 EUR';", "price", "155 EUR"},
		{"dLayer['distinction'] = '';", "distinction", ""},
		{"dLayer = { 'distinction': '1 star' };", "distinction", ""},
		{"", "distinction", ""},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := ParseDLayerValue(tt.script, tt.key)
			if got != tt.expected {
				t.Errorf("ParseDLayerValue(%q, %q) = %q; want %q", tt.script, tt.key, got, tt.expected)
			}
		})
	}
}
