package parsers

import "testing"

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
			got := ExtractPhoneNumber(tt.input)
			if got != tt.expected {
				t.Errorf("ParsePhoneNumber(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
