package parsers

import "testing"

func TestExtractOriginalURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://web.archive.org/web/20230101120000/https://guide.michelin.com/en/restaurants",
			"https://guide.michelin.com/en/restaurants",
		},
		{
			// wayback URL with modifier flag (if_) after the timestamp
			"https://web.archive.org/web/20210615083045if_/https://guide.michelin.com/en/restaurant/foo",
			"https://guide.michelin.com/en/restaurant/foo",
		},
		{
			// not a wayback URL – returned as-is
			"https://guide.michelin.com/en/restaurants",
			"https://guide.michelin.com/en/restaurants",
		},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractOriginalURL(tt.input)
			if got != tt.expected {
				t.Errorf("extractOriginalURL(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
