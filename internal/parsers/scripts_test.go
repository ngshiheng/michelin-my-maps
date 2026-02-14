package parsers

import (
	"testing"
)

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
			got := parseDLayerValue(tt.script, tt.key)
			if got != tt.expected {
				t.Errorf("parseDLayerValue(%q, %q) = %q; want %q", tt.script, tt.key, got, tt.expected)
			}
		})
	}
}
