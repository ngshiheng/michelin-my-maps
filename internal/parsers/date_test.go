package parsers

import (
	"testing"
	"time"
)

func TestParseDateFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2023 MICHELIN Guide", "2023"},
		{"MICHELIN Guide Singapore 2022", "2022"},
		{"2021-07-15", "2021-07-15"},
		{"No date here", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDateFromText(tt.input)
			if got != tt.expected {
				t.Errorf("parseDateFromText(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseYearFromAnyFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"2023-01-25", 2023},
		{"2022 MICHELIN Guide", 2022},
		{"MICHELIN Guide Singapore 2021", 2021},
		{"2019", 2019},
		{"", 0},
		{"not a date", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseYearFromAnyFormat(tt.input)
			if got != tt.expected {
				t.Errorf("parseYearFromAnyFormat(%q) = %d; want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParsePublishedYear(t *testing.T) {
	tests := []struct {
		name     string
		jsonLD   string
		expected int
	}{
		{
			name:     "review.datePublished full date",
			jsonLD:   `{"review":{"datePublished":"2023-01-25"}}`,
			expected: 2023,
		},
		{
			name:     "review.datePublished year only",
			jsonLD:   `{"review":{"datePublished":"2019"}}`,
			expected: 2019,
		},
		{
			name:     "review.datePublished invalid",
			jsonLD:   `{"review":{"datePublished":"not-a-date"}}`,
			expected: 0,
		},
		{
			name:     "missing review",
			jsonLD:   `{}`,
			expected: 0,
		},
		{
			name:     "award.dateAwarded as 4-digit year",
			jsonLD:   `{"award":{"dateAwarded":"2022"}}`,
			expected: 2022,
		},
		{
			name:     "award.dateAwarded as ISO date",
			jsonLD:   `{"award":{"dateAwarded":"2021-07-15"}}`,
			expected: 2021,
		},
		{
			name:     "award prioritized over review",
			jsonLD:   `{"award":{"dateAwarded":"2020"},"review":{"datePublished":"2019"}}`,
			expected: 2020,
		},
		{
			name:     "award invalid, fallback to review",
			jsonLD:   `{"award":{"dateAwarded":"not-a-date"},"review":{"datePublished":"2018"}}`,
			expected: 2018,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year := parsePublishedYear(tt.jsonLD)
			if year != tt.expected {
				t.Errorf("ParsePublishedYear(%s) = %d; want %d", tt.name, year, tt.expected)
			}
		})
	}
}

func TestValidateYear(t *testing.T) {
	currentYear := time.Now().Year()
	tests := []struct {
		input    int
		expected bool
	}{
		{1900, true},
		{currentYear, true},
		{currentYear + 1, true},
		{1899, false},
		{currentYear + 2, false},
		{0, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := validateYear(tt.input)
			if got != tt.expected {
				t.Errorf("validateYear(%d) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}
