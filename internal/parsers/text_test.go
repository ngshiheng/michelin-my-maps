package parsers

import (
	"testing"
)

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Shaw Centre, #01-16,\n1 Scotts Road, 228208, Singapore", "Shaw Centre, #01-16, 1 Scotts Road, 228208, Singapore"},
		{"123 Main St\nNew York", "123 Main St New York"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeAddress(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeAddress(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTrimWhiteSpaces(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Hello  World\n", "Hello World"},
		{"\n\nTest\n", "Test"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TrimWhiteSpaces(tt.input)
			if got != tt.expected {
				t.Errorf("TrimWhiteSpaces(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractTextFromElements(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{"", "  ", "First", "Second"}, "First"},
		{[]string{"", "  ", ""}, ""},
		{[]string{"Only"}, "Only"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ExtractTextFromElements(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractTextFromElements(%v) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestJoinFacilities(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{"Air conditioning", "", "Car park", "Interesting wine list"}, "Air conditioning,Car park,Interesting wine list"},
		{[]string{"", ""}, ""},
		{[]string{"WiFi"}, "WiFi"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := JoinFacilities(tt.input)
			if got != tt.expected {
				t.Errorf("JoinFacilities(%v) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitUnpack(t *testing.T) {
	tests := []struct {
		input     string
		separator string
		expectedA string
		expectedB string
	}{
		{"price: 100", ":", "price", "100"},
		{"no-separator", ":", "", "no-separator"},
		{"", ":", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			a, b := SplitUnpack(tt.input, tt.separator)
			if a != tt.expectedA || b != tt.expectedB {
				t.Errorf("SplitUnpack(%q, %q) = (%q, %q); want (%q, %q)", tt.input, tt.separator, a, b, tt.expectedA, tt.expectedB)
			}
		})
	}
}
